package entities

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Release struct {
	ProjectName   string
	Number        int
	Port          int
	ID            string
	Cmd           *exec.Cmd
	Done          chan struct{}
	alive         atomic.Bool
	active        atomic.Bool
	stopRequested atomic.Bool
}

type ReleaseRuntime struct {
	Release int  `json:"release"`
	PID     int  `json:"pid,omitempty"`
	Alive   bool `json:"alive"`
	Active  bool `json:"active"`
}

var (
	releasesMu   sync.RWMutex
	releases     = make(map[string]*Release)
	shuttingDown atomic.Bool
)

func registerRelease(release *Release) {
	releasesMu.Lock()
	releases[release.ID] = release
	releasesMu.Unlock()
}

func unregisterRelease(release *Release) {
	releasesMu.Lock()
	if current, ok := releases[release.ID]; ok && current == release {
		delete(releases, release.ID)
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
	releasesMu.RLock()
	release, ok := releases[releaseID(projectName, releaseNumber)]
	releasesMu.RUnlock()
	return release, ok
}

func ReleaseRuntimeStatus(projectName string, releaseNumber int) ReleaseRuntime {
	status := ReleaseRuntime{Release: releaseNumber}
	release, ok := getRelease(projectName, releaseNumber)
	if !ok {
		return status
	}
	status.Alive = release.alive.Load()
	status.Active = release.active.Load()
	if release.Cmd.Process != nil {
		status.PID = release.Cmd.Process.Pid
	}
	return status
}

func requestReleaseStop(release *Release) {
	release.stopRequested.Store(true)
	if release.Cmd.Process != nil {
		err := syscall.Kill(-release.Cmd.Process.Pid, syscall.SIGTERM)
		if err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
			slog.Info("Failed to signal release", "release", release.ID, "err", err)
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

func releaseID(projectName string, releaseNumber int) string {
	return fmt.Sprintf("%s-r%d", projectName, releaseNumber)
}

func (project *Project) defaultDeployCommands() []string {
	image := project.DockerImage()
	return []string{fmt.Sprintf(`docker build -t %s .`, image), fmt.Sprintf(`docker run -p "$PORT":80 --rm %s`, image)}
}

func (project *Project) deployCommand() ([]string, error) {
	if project.Additional == nil {
		project.Additional = make(map[string]interface{})
	}
	value, exists := project.Additional["cmd"]
	if exists {
		var commands []string
		switch values := value.(type) {
		case []string:
			commands = values
		case []interface{}:
			for _, value := range values {
				command, ok := value.(string)
				if !ok {
					return nil, errors.New("additional.cmd must contain only strings")
				}
				commands = append(commands, command)
			}
		default:
			return nil, errors.New("additional.cmd must be an array of strings")
		}
		result := make([]string, 0, len(commands))
		for _, command := range commands {
			if strings.TrimSpace(command) != "" {
				result = append(result, command)
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}
	commands := project.defaultDeployCommands()
	project.Additional["cmd"] = commands
	return commands, nil
}

func releaseEnvironment(port int) []string {
	environment := os.Environ()
	result := make([]string, 0, len(environment)+1)
	for _, variable := range environment {
		if !strings.HasPrefix(variable, "PORT=") {
			result = append(result, variable)
		}
	}
	return append(result, "PORT="+strconv.Itoa(port))
}

func (project *Project) BuildAndRun() error {
	return project.buildAndRun(nil)
}

func (project *Project) buildAndRun(ready chan<- struct{}) error {
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
	releaseID := project.ReleaseID(releaseNumber)
	commands, err := project.deployCommand()
	if err != nil {
		project.State = fmt.Sprintf("Error: %v", err)
		_ = project.Save()
		return err
	}
	domains, err := project.ProjectDomains()
	if err != nil {
		return err
	}
	hasDomain := len(domains) > 0

	project.State = "starting"
	if !hasDomain {
		project.State = "cleanup"
	}
	if err := project.Save(); err != nil {
		return err
	}
	if dry {
		slog.Info("Executing", "cmd", commands, "dir", releaseDir)
		if ready != nil {
			close(ready)
		}
		if !hasDomain {
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
	for _, command := range commands[:len(commands)-1] {
		_, _ = fmt.Fprintf(logFile, "\n--- %s ---\n$ %s\n", time.Now().Format(time.RFC3339), command)
		step := exec.Command("sh", "-c", command)
		step.Dir, step.Env, step.Stdout, step.Stderr = releaseDir, releaseEnvironment(project.Port), logFile, logFile
		if err := step.Run(); err != nil {
			_ = logFile.Close()
			project.State = fmt.Sprintf("Error: %v", err)
			_ = project.Save()
			return err
		}
	}
	command := commands[len(commands)-1]
	_, _ = fmt.Fprintf(logFile, "\n--- %s ---\n$ %s\n", time.Now().Format(time.RFC3339), command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = releaseDir
	cmd.Env = releaseEnvironment(project.Port)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		project.State = fmt.Sprintf("Error: %v", err)
		return project.Save()
	}

	release := &Release{
		ProjectName: project.Name,
		Number:      releaseNumber,
		Port:        project.Port,
		ID:          releaseID,
		Cmd:         cmd,
		Done:        make(chan struct{}),
	}
	release.alive.Store(true)
	registerRelease(release)
	go waitForRelease(release, logFile)

	if !hasDomain {
		if err := project.advance(false); err != nil {
			if ready != nil {
				close(ready)
			}
			return err
		}
		release.active.Store(true)
		if ready != nil {
			close(ready)
		}
		return nil
	}

	go func() {
		coordinateReleasePromotion(release)
		if ready != nil {
			close(ready)
		}
	}()
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
}

func coordinateReleasePromotion(release *Release) {
	if !release.alive.Load() {
		return
	}
	err := promoteRelease(release)
	if err != nil && release.alive.Load() {
		project, projectErr := GetProject(release.ProjectName)
		if projectErr != nil {
			return
		}
		current, currentErr := project.CurrentReleaseNumber()
		if currentErr != nil || current != release.Number {
			return
		}
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
	if project.State != "starting" {
		return fmt.Errorf("project is not ready to promote: %s", project.State)
	}

	select {
	case <-release.Done:
		return errors.New("release exited during startup grace period")
	case <-time.After(5 * time.Second):
	}
	if shuttingDown.Load() || !release.alive.Load() {
		return errors.New("release is not alive after startup grace period")
	}

	project.State = "starting"
	if err := project.Save(); err != nil {
		return err
	}
	if err := project.Proxy(); err != nil {
		return rollbackPromotion(&project, release, err)
	}
	if !release.alive.Load() {
		return rollbackPromotion(&project, release, errors.New("release exited during proxy setup"))
	}
	if project.IsError() {
		return rollbackPromotion(&project, release, errors.New(project.State))
	}
	if err := project.Cert(); err != nil {
		return rollbackPromotion(&project, release, err)
	}
	if !release.alive.Load() {
		return rollbackPromotion(&project, release, errors.New("release exited during certificate setup"))
	}
	if project.IsError() {
		return rollbackPromotion(&project, release, errors.New(project.State))
	}
	if err := project.Cleanup(); err != nil {
		return savePromotionError(&project, release.Number, err)
	}
	release.active.Store(true)
	return nil
}

func rollbackPromotion(project *Project, release *Release, err error) error {
	if rollbackErr := restorePreviousProxy(project, release.Number); rollbackErr != nil {
		err = fmt.Errorf("%v; proxy rollback failed: %w", err, rollbackErr)
	}
	return savePromotionError(project, release.Number, err)
}

func restorePreviousProxy(project *Project, currentRelease int) error {
	domains, err := project.ProjectDomains()
	if err != nil || len(domains) == 0 {
		return err
	}
	var previous *Release
	for _, release := range projectReleases(project.Name, false, currentRelease) {
		if release.active.Load() && release.alive.Load() && !release.stopRequested.Load() && (previous == nil || release.Number > previous.Number) {
			previous = release
		}
	}
	if previous == nil {
		return nil
	}
	return project.retargetProxy(previous.Port)
}

func savePromotionError(project *Project, releaseNumber int, err error) error {
	project.State = fmt.Sprintf("Error: failed to promote release r%d: %v", releaseNumber, err)
	if saveErr := project.Save(); saveErr != nil {
		slog.Info("Failed to save promotion failure", "project", project.Name, "err", saveErr)
	}
	return err
}
