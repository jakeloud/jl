package entities

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Release struct {
	ProjectName   string
	Number        int
	Port          int
	ContainerName string
	Cmd           *exec.Cmd
	Done          chan struct{}
	Promote       chan promotionRequest
	PromotionDone chan struct{}
	PromoteAt     time.Time
	alive         atomic.Bool
	stopRequested atomic.Bool
}

type promotionRequest struct {
	result chan error
}

var (
	releasesMu             sync.RWMutex
	releases               = make(map[string]*Release)
	shuttingDown           atomic.Bool
	notifierMu             sync.RWMutex
	releaseFailureNotifier func(string) error
)

func SetReleaseFailureNotifier(notifier func(string) error) {
	notifierMu.Lock()
	releaseFailureNotifier = notifier
	notifierMu.Unlock()
}

func notifyReleaseFailure(message string) {
	notifierMu.RLock()
	notifier := releaseFailureNotifier
	notifierMu.RUnlock()
	if notifier != nil {
		_ = notifier(message)
	}
}

func registerRelease(release *Release) {
	releasesMu.Lock()
	releases[release.ContainerName] = release
	releasesMu.Unlock()
}

func unregisterRelease(release *Release) {
	releasesMu.Lock()
	if current, ok := releases[release.ContainerName]; ok && current == release {
		delete(releases, release.ContainerName)
	}
	releasesMu.Unlock()
}

func projectReleases(projectName string, includeCurrent bool, current int) []*Release {
	releasesMu.RLock()
	defer releasesMu.RUnlock()
	result := make([]*Release, 0)
	for _, release := range releases {
		if (projectName == "" || release.ProjectName == projectName) && (includeCurrent || release.Number != current) {
			result = append(result, release)
		}
	}
	return result
}

func reservedReleasePorts() map[int]bool {
	releasesMu.RLock()
	defer releasesMu.RUnlock()
	ports := make(map[int]bool, len(releases))
	for _, release := range releases {
		ports[release.Port] = true
	}
	return ports
}

func getRelease(projectName string, releaseNumber int) (*Release, bool) {
	containerName := fmt.Sprintf("%s-r%d", projectName, releaseNumber)
	releasesMu.RLock()
	release, ok := releases[containerName]
	releasesMu.RUnlock()
	return release, ok
}

func ReleasePromotionDeadline(projectName string, releaseNumber int) (time.Time, bool) {
	release, ok := getRelease(projectName, releaseNumber)
	if !ok || release.PromoteAt.IsZero() || !release.alive.Load() {
		return time.Time{}, false
	}
	select {
	case <-release.PromotionDone:
		return time.Time{}, false
	default:
	}
	return release.PromoteAt, true
}

func requestReleaseStop(release *Release) {
	release.stopRequested.Store(true)
	if release.Cmd.Process != nil {
		err := syscall.Kill(-release.Cmd.Process.Pid, syscall.SIGTERM)
		if err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
			slog.Info("Failed to signal release", "container", release.ContainerName, "err", err)
		}
	}
}

func waitForReleases(releaseList []*Release, timeout time.Duration) bool {
	if len(releaseList) == 0 {
		return true
	}

	done := make(chan struct{})
	go func() {
		for _, release := range releaseList {
			<-release.Done
		}
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func shutdownReleaseList(releaseList []*Release, timeout time.Duration) bool {
	for _, release := range releaseList {
		requestReleaseStop(release)
	}
	return waitForReleases(releaseList, timeout)
}

func ShutdownReleases(timeout time.Duration) bool {
	shuttingDown.Store(true)
	return shutdownReleaseList(projectReleases("", true, 0), timeout)
}

func BeginShutdown() {
	shuttingDown.Store(true)
}

func (project *Project) BuildAndRun() error {
	if shuttingDown.Load() {
		return errors.New("jakeloud is shutting down")
	}
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "cloning" {
		return nil
	}
	releaseDir, err := project.CurrentReleaseDir()
	if err != nil {
		return err
	}
	releaseNumber, err := project.CurrentReleaseNumber()
	if err != nil {
		return err
	}
	containerName := project.ReleaseContainerName(releaseNumber)
	dockerOptions := ""
	if opts, ok := project.Additional["dockerOptions"].(string); ok {
		dockerOptions = opts
	}
	domain, delay, err := ParseProjectDomain(project.Domain)
	if err != nil {
		return err
	}

	command := fmt.Sprintf(`docker build -t %s . && exec docker run --rm --sig-proxy=true --name %s -p %d:80 %s %s`, project.DockerImage(), containerName, project.Port, dockerOptions, project.DockerImage())
	project.State = "awaiting liveness"
	if domain == "" {
		project.State = "cleanup"
	}
	if err := project.Save(); err != nil {
		return err
	}
	if dry {
		slog.Info("Executing", "cmd", command, "dir", releaseDir)
		if domain == "" {
			return project.advance(false)
		}
		return nil
	}
	if shuttingDown.Load() {
		project.State = "Error: jakeloud shut down before release launch"
		_ = project.Save()
		return errors.New("jakeloud is shutting down")
	}

	logFile, err := os.OpenFile(project.ReleaseLogPath(releaseNumber), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		project.State = fmt.Sprintf("Error: %v", err)
		_ = project.Save()
		return err
	}
	_, _ = fmt.Fprintf(logFile, "\n--- %s ---\n$ %s\n", time.Now().Format(time.RFC3339), command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = releaseDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", project.Port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		project.State = fmt.Sprintf("Error: %v", err)
		return project.Save()
	}

	release := &Release{
		ProjectName:   project.Name,
		Number:        releaseNumber,
		Port:          project.Port,
		ContainerName: containerName,
		Cmd:           cmd,
		Done:          make(chan struct{}),
		Promote:       make(chan promotionRequest),
		PromotionDone: make(chan struct{}),
	}
	release.alive.Store(true)
	if domain != "" {
		release.PromoteAt = time.Now().Add(delay)
	}
	registerRelease(release)
	go waitForRelease(release, logFile)

	if domain == "" {
		close(release.PromotionDone)
		return project.advance(false)
	}

	go coordinateReleasePromotion(release, delay)
	return nil
}

func waitForRelease(release *Release, logFile *os.File) {
	err := release.Cmd.Wait()
	release.alive.Store(false)
	_ = logFile.Close()
	unregisterRelease(release)
	close(release.Done)

	if release.stopRequested.Load() || shuttingDown.Load() {
		return
	}

	lock := projectLock(release.ProjectName)
	lock.Lock()
	defer lock.Unlock()
	project, getErr := GetProject(release.ProjectName)
	if getErr != nil {
		return
	}
	current, currentErr := project.CurrentReleaseNumber()
	if currentErr != nil || current != release.Number {
		return
	}
	if err == nil {
		err = errors.New("release process exited")
	}
	project.State = fmt.Sprintf("Error: release r%d exited: %v", release.Number, err)
	if saveErr := project.Save(); saveErr != nil {
		slog.Info("Failed to save release failure", "project", release.ProjectName, "err", saveErr)
	}
	notifyReleaseFailure(fmt.Sprintf("*%s* release r%d failed: %v", release.ProjectName, release.Number, err))
}

func coordinateReleasePromotion(release *Release, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	defer close(release.PromotionDone)

	var request *promotionRequest
	select {
	case <-release.Done:
		return
	case received := <-release.Promote:
		request = &received
	case <-timer.C:
	}

	err := promoteRelease(release)
	if request != nil {
		request.result <- err
	}
	if err != nil && release.alive.Load() {
		project, projectErr := GetProject(release.ProjectName)
		if projectErr != nil {
			return
		}
		current, currentErr := project.CurrentReleaseNumber()
		if currentErr != nil || current != release.Number {
			return
		}
		notifyReleaseFailure(fmt.Sprintf("*%s* release r%d promotion failed: %v", release.ProjectName, release.Number, err))
	}
}

func promoteRelease(release *Release) error {
	lock := projectLock(release.ProjectName)
	lock.Lock()
	defer lock.Unlock()

	if shuttingDown.Load() || !release.alive.Load() {
		return errors.New("release is not alive")
	}
	registered, ok := getRelease(release.ProjectName, release.Number)
	if !ok || registered != release {
		return errors.New("release is no longer registered")
	}

	project, err := GetProject(release.ProjectName)
	if err != nil {
		return err
	}
	current, err := project.CurrentReleaseNumber()
	if err != nil {
		return err
	}
	if current != release.Number {
		return errors.New("release has been superseded")
	}
	if project.State != "awaiting liveness" {
		return fmt.Errorf("project is not awaiting liveness: %s", project.State)
	}

	project.State = "starting"
	if err := project.Save(); err != nil {
		return err
	}
	if err := project.advance(false); err != nil {
		project.State = fmt.Sprintf("Error: failed to promote release r%d: %v", release.Number, err)
		if saveErr := project.Save(); saveErr != nil {
			slog.Info("Failed to save promotion failure", "project", project.Name, "err", saveErr)
		}
		return err
	}
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.IsError() {
		return errors.New(project.State)
	}
	return nil
}

func ConfirmRelease(projectName string, releaseNumber int) error {
	project, err := GetProject(projectName)
	if err != nil {
		return err
	}
	domain, _, err := ParseProjectDomain(project.Domain)
	if err != nil {
		return err
	}
	if domain == "" {
		return errors.New("project does not use a domain")
	}
	if project.State != "awaiting liveness" {
		return fmt.Errorf("project is not awaiting liveness: %s", project.State)
	}

	release, ok := getRelease(projectName, releaseNumber)
	if !ok || !release.alive.Load() {
		return errors.New("release is not alive")
	}
	request := promotionRequest{result: make(chan error, 1)}
	select {
	case release.Promote <- request:
	case <-release.Done:
		return errors.New("release exited before confirmation")
	case <-release.PromotionDone:
		return errors.New("release promotion has already completed")
	}

	select {
	case err := <-request.result:
		return err
	case <-release.Done:
		return errors.New("release exited during promotion")
	}
}
