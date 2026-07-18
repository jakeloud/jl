package entities

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jakeloud/jl/ip_getter"
	"golang.org/x/crypto/pbkdf2"
)

const (
	PROJECTS_ROOT = "/app"
	JAKELOUD      = "jakeloud"
	LOG_MUTEX     = false
)

var CONF_FILE = PROJECTS_ROOT + "/conf.json"
var SSH_KEY = PROJECTS_ROOT + "/id_rsa"
var SSH_KEY_PUB = PROJECTS_ROOT + "/id_rsa.pub"

var dry bool = false
var dry_conf []byte = []byte("{\"apps\":[{\"name\":\"jakeloud\",\"port\":666}],\"users\":[]}")

type Release struct {
	ProjectName   string
	Number        int
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
	confMu                 sync.Mutex
	projectLocksMu         sync.Mutex
	projectLocks           = make(map[string]*sync.Mutex)
)

const defaultProxyDelay = 5 * time.Minute

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

func SetDry(d bool) {
	dry = d
}

type Project struct {
	Name       string                 `json:"name"`
	Domain     string                 `json:"domain,omitempty"`
	Repo       string                 `json:"repo,omitempty"`
	Port       int                    `json:"port,omitempty"`
	State      string                 `json:"state,omitempty"`
	Email      string                 `json:"email,omitempty"`
	Additional map[string]interface{} `json:"additional,omitempty"`
	mu         sync.Mutex
}

type Config struct {
	Projects []Project `json:"apps"`
	Users    []User    `json:"users"`
}

type User struct {
	Email string `json:"email"`
	Hash  []byte `json:"hash"`
	Salt  string `json:"salt"`
}

func SetConf(conf Config) error {
	confMu.Lock()
	defer confMu.Unlock()
	return setConf(conf)
}

func setConf(conf Config) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}

	if dry {
		dry_conf = data
		return nil
	}

	return ioutil.WriteFile(CONF_FILE, data, 0644)
}

func GetConf() (Config, error) {
	confMu.Lock()
	defer confMu.Unlock()
	return getConf()
}

func getConf() (Config, error) {
	var conf Config
	if dry {
		if err := json.Unmarshal(dry_conf, &conf); err != nil {
			return conf, err
		}
		return conf, nil
	}

	data, err := ioutil.ReadFile(CONF_FILE)
	if err != nil {
		fmt.Printf("Problem with conf.json: %v\n", err)
		conf = Config{
			Projects: []Project{{Name: JAKELOUD, Port: 666}},
			Users:    []User{},
		}
		return conf, nil
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

func projectLock(name string) *sync.Mutex {
	projectLocksMu.Lock()
	defer projectLocksMu.Unlock()
	lock, ok := projectLocks[name]
	if !ok {
		lock = &sync.Mutex{}
		projectLocks[name] = lock
	}
	return lock
}

func ExecWrapped(cmd string) (string, error) {
	if dry {
		slog.Info("Executing", "cmd", cmd)
		return "", nil
	}

	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func (project *Project) ReleaseLogPath(releaseNumber int) string {
	return filepath.Join(project.ProjectDir(), fmt.Sprintf("r%d.log", releaseNumber))
}

func (project *Project) runReleaseCommand(releaseNumber int, command string) (string, error) {
	if dry {
		slog.Info("Executing", "cmd", command)
		return "", nil
	}

	if err := os.MkdirAll(project.ProjectDir(), 0755); err != nil {
		return "", err
	}
	logFile, err := os.OpenFile(project.ReleaseLogPath(releaseNumber), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer logFile.Close()

	_, _ = fmt.Fprintf(logFile, "\n--- %s ---\n$ %s\n", time.Now().Format(time.RFC3339), command)
	output, err := exec.Command("sh", "-c", command).CombinedOutput()
	if _, writeErr := logFile.Write(output); writeErr != nil && err == nil {
		err = writeErr
	}
	return string(output), err
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
	if release.stopRequested.CompareAndSwap(false, true) && release.Cmd.Process != nil {
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

func ClearCache() (string, error) {
	if dry {
		slog.Info("Clearing cache")
		return "", nil
	}

	res, err := ExecWrapped("docker system prune -af")
	if err != nil {
		return err.Error(), err
	}
	return res, nil
}

func Start(server interface{}) error {
	project, err := GetProject(JAKELOUD)
	if err != nil {
		return err
	}

	if !dry {
		if _, err := os.Stat(SSH_KEY); os.IsNotExist(err) {
			_, err := ExecWrapped(`ssh-keygen -q -t ed25519 -N '' -f ` + SSH_KEY)
			if err != nil {
				return err
			}
		}

		sshKey, err := ioutil.ReadFile(SSH_KEY_PUB)
		if err != nil {
			return err
		}

		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		if project.Additional == nil {
			project.Additional = make(map[string]interface{})
		}
		project.Additional["sshKey"] = string(sshKey)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}

		if err := project.Save(); err != nil {
			return err
		}
	}

	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "starting"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}

	if project.Domain == "" {
		dom, err := ip_getter.GetPublicIP()
		if err != nil {
			slog.Info("Failed to get ip", "err", err)
			return err
		}
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.Domain = fmt.Sprintf("jakeloud.%s.sslip.io", dom)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
	}

	if err := project.Save(); err != nil {
		return err
	}
	if err := project.Proxy(); err != nil {
		return err
	}
	if err := project.LoadState(); err != nil {
		return err
	}
	if err := project.Cert(); err != nil {
		return err
	}

	go redeployProjects()
	return nil
}

func redeployProjects() {
	conf, err := GetConf()
	if err != nil {
		slog.Info("Failed to load projects for startup redeploy", "err", err)
		return
	}
	for _, project := range conf.Projects {
		if project.Name == JAKELOUD {
			continue
		}
		if err := project.Advance(true); err != nil {
			slog.Info("Startup redeploy failed", "project", project.Name, "err", err)
		}
	}
}

func (project *Project) Save() error {
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	defer project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}

	confMu.Lock()
	defer confMu.Unlock()
	conf, err := getConf()
	if err != nil {
		return err
	}

	projectIndex := -1
	for i, p := range conf.Projects {
		if p.Name == project.Name {
			projectIndex = i
			break
		}
	}

	if projectIndex == -1 {
		conf.Projects = append(conf.Projects, *project)
	} else {
		conf.Projects[projectIndex] = *project
	}

	return setConf(conf)
}

func (project *Project) AllocatePortAndSave() error {
	confMu.Lock()
	defer confMu.Unlock()

	conf, err := getConf()
	if err != nil {
		return err
	}
	takenPorts := make(map[int]bool)
	for _, configuredProject := range conf.Projects {
		takenPorts[configuredProject.Port] = true
	}
	project.Port = 38000
	for takenPorts[project.Port] {
		project.Port++
	}

	for i, configuredProject := range conf.Projects {
		if configuredProject.Name == project.Name {
			conf.Projects[i] = *project
			return setConf(conf)
		}
	}
	conf.Projects = append(conf.Projects, *project)
	return setConf(conf)
}

func (project *Project) DeployWithNewPort() error {
	lock := projectLock(project.Name)
	lock.Lock()
	defer lock.Unlock()
	if err := project.AllocatePortAndSave(); err != nil {
		return err
	}
	return project.advance(true)
}

func (project *Project) ProjectDir() string {
	return filepath.Join(PROJECTS_ROOT, project.Name)
}

func (project *Project) DockerImage() string {
	return strings.ToLower(project.Name)
}

func ParseProjectDomain(value string) (string, time.Duration, error) {
	if value == "" {
		return "", 0, nil
	}
	if strings.Contains(value, "://") || strings.ContainsAny(value, "/?# 	\r\n") {
		return "", 0, fmt.Errorf("invalid project domain %q", value)
	}

	host := value
	delay := defaultProxyDelay
	if separator := strings.LastIndex(value, ":"); separator >= 0 {
		host = value[:separator]
		minutes, err := strconv.Atoi(value[separator+1:])
		if err != nil || minutes < 1 || minutes > 525600 {
			return "", 0, fmt.Errorf("invalid proxy delay in domain %q", value)
		}
		delay = time.Duration(minutes) * time.Minute
	}
	if host == "" || strings.Contains(host, ":") {
		return "", 0, fmt.Errorf("invalid project domain %q", value)
	}
	return host, delay, nil
}

func (project *Project) DomainSettings() (string, time.Duration, error) {
	return ParseProjectDomain(project.Domain)
}

func (project *Project) ReleaseContainerName(releaseNumber int) string {
	return fmt.Sprintf("%s-r%d", project.Name, releaseNumber)
}

func releaseNumber(name string) (int, bool) {
	if !strings.HasPrefix(name, "r") {
		return 0, false
	}

	n, err := strconv.Atoi(strings.TrimPrefix(name, "r"))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func (project *Project) ReleaseDirs() (map[int]string, error) {
	releases := make(map[int]string)
	entries, err := os.ReadDir(project.ProjectDir())
	if err != nil {
		if os.IsNotExist(err) {
			return releases, nil
		}
		return releases, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		n, ok := releaseNumber(entry.Name())
		if !ok {
			continue
		}
		releases[n] = filepath.Join(project.ProjectDir(), entry.Name())
	}
	return releases, nil
}

func (project *Project) CurrentReleaseDir() (string, error) {
	current, releases, err := project.CurrentRelease()
	if err != nil {
		return "", err
	}
	return releases[current], nil
}

func (project *Project) CurrentReleaseNumber() (int, error) {
	current, _, err := project.CurrentRelease()
	return current, err
}

func (project *Project) CurrentContainerName() (string, error) {
	current, err := project.CurrentReleaseNumber()
	if err != nil {
		return "", err
	}
	return project.ReleaseContainerName(current), nil
}

func (project *Project) CurrentRelease() (int, map[int]string, error) {
	releases, err := project.ReleaseDirs()
	if err != nil {
		return 0, releases, err
	}

	current := 0
	for n := range releases {
		if n > current {
			current = n
		}
	}
	if current == 0 {
		return 0, releases, errors.New("release not found")
	}
	return current, releases, nil
}

func (project *Project) NewRelease() (string, error) {
	if err := os.MkdirAll(project.ProjectDir(), 0755); err != nil {
		return "", err
	}

	releases, err := project.ReleaseDirs()
	if err != nil {
		return "", err
	}

	current := 0
	for n := range releases {
		if n > current {
			current = n
		}
	}

	next := current + 1
	releaseDir := filepath.Join(project.ProjectDir(), fmt.Sprintf("r%d", next))
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		return "", err
	}

	keep := map[int]bool{next: true, next - 1: true}
	oldReleaseNumbers := make([]int, 0)
	for n := range releases {
		if !keep[n] {
			oldReleaseNumbers = append(oldReleaseNumbers, n)
		}
	}
	sort.Ints(oldReleaseNumbers)
	for _, n := range oldReleaseNumbers {
		if err := os.RemoveAll(releases[n]); err != nil {
			return "", err
		}
	}

	return releaseDir, nil
}

func (project *Project) LoadState() error {
	loadedProject, err := GetProject(project.Name)
	if err != nil {
		return err
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = loadedProject.State
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	return nil
}

func (project *Project) Clone() error {
	slog.Info("Cloning", "project", project.Name)
	if err := project.LoadState(); err != nil {
		return err
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "cloning"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	releaseDir, err := project.NewRelease()
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = fmt.Sprintf("Error: %v", err)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}

	releaseNumberValue, ok := releaseNumber(filepath.Base(releaseDir))
	if !ok {
		return fmt.Errorf("invalid release directory: %s", releaseDir)
	}
	cmd := fmt.Sprintf(`eval "$(ssh-agent -s)"; ssh-add %s; git clone --depth 1 %s %s; kill $SSH_AGENT_PID`, SSH_KEY, project.Repo, releaseDir)
	out, err := project.runReleaseCommand(releaseNumberValue, cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = fmt.Sprintf("Error: %v\n%s", err, out)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}
	return nil
}

func (project *Project) BuildAndRun() error {
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
	domain, delay, err := project.DomainSettings()
	if err != nil {
		return err
	}

	command := fmt.Sprintf(`docker build -t %s . && exec docker run --rm --sig-proxy=true --name %s -p %d:80 %s %s`, project.DockerImage(), containerName, project.Port, dockerOptions, project.DockerImage())
	if dry {
		slog.Info("Executing", "cmd", command, "dir", releaseDir)
		if domain == "" {
			project.State = "cleanup"
			if err := project.Save(); err != nil {
				return err
			}
			return project.advance(false)
		}
		project.State = "awaiting liveness"
		return project.Save()
	}

	logFile, err := os.OpenFile(project.ReleaseLogPath(releaseNumber), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
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
		project.State = "cleanup"
		if err := project.Save(); err != nil {
			requestReleaseStop(release)
			return err
		}
		return project.advance(false)
	}

	project.State = "awaiting liveness"
	if err := project.Save(); err != nil {
		requestReleaseStop(release)
		return err
	}
	go coordinateReleasePromotion(release, delay)
	return nil
}

func (project *Project) Proxy() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "starting" {
		return nil
	}
	domain, _, err := project.DomainSettings()
	if err != nil {
		return err
	}
	if domain == "" {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = "cleanup"
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "proxying"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	content := fmt.Sprintf(`
server {
	listen 80;
	server_name %s;

	location / {
                client_max_body_size 100M;
		proxy_set_header   X-Forwarded-For $remote_addr;
		proxy_set_header   Host $host;
		proxy_pass         http://127.0.0.1:%d;

		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
	}
}`, domain, project.Port)

	file := "default"
	if project.Name != JAKELOUD {
		file = project.Name
	}
	filePath := fmt.Sprintf("/etc/nginx/sites-available/%s", file)

	if dry {
		slog.Info("Writing", "file", filePath, "content", content)
	} else {
		if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	if out, err := ExecWrapped("nginx -t"); err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = fmt.Sprintf("Error: %v\n%s", err, out)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}

	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s", file)
	if dry {
		slog.Info("Enabling", "path", enabledPath)
	} else {
		if _, err := os.Stat(enabledPath); os.IsNotExist(err) {
			if err := os.Symlink(filePath, enabledPath); err != nil {
				return err
			}
		}
	}

	if out, err := ExecWrapped("service nginx restart"); err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = fmt.Sprintf("Error: %v\n%s", err, out)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}
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
	domain, _, err := project.DomainSettings()
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

func (project *Project) Cert() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "proxying" {
		return nil
	}
	domain, _, err := project.DomainSettings()
	if err != nil {
		return err
	}
	if domain == "" {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = "cleanup"
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "certing"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	email := project.Email
	if email == "" {
		email = "no-reply@gmail.com"
	}
	cmd := fmt.Sprintf(`certbot -n --agree-tos --email %s --nginx -d %s`, email, domain)
	out, err := ExecWrapped(cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "project", project.Name)
		}
		project.mu.Lock()
		project.State = fmt.Sprintf("Error: %v\n%s", err, out)
		project.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "project", project.Name)
		}
		return project.Save()
	}

	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	nextState := "cleanup"
	if project.Name == JAKELOUD {
		nextState = "🟢 running"
	}
	project.mu.Lock()
	project.State = nextState
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	return project.Save()
}

func (project *Project) Cleanup() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "cleanup" {
		return nil
	}

	current, err := project.CurrentReleaseNumber()
	if err != nil {
		return err
	}
	oldReleases := projectReleases(project.Name, false, current)
	if !shutdownReleaseList(oldReleases, 5*time.Second) {
		err := fmt.Errorf("timed out waiting for previous releases to stop")
		project.State = fmt.Sprintf("Error: %v", err)
		if saveErr := project.Save(); saveErr != nil {
			return saveErr
		}
		notifyReleaseFailure(fmt.Sprintf("*%s* cleanup failed: %v", project.Name, err))
		return err
	}

	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "🟢 running"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	return project.Save()
}

func (project *Project) Stop() error {
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "stopping"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	if _, err := project.CurrentReleaseNumber(); err != nil {
		return err
	}
	if !shutdownReleaseList(projectReleases(project.Name, true, 0), 5*time.Second) {
		return fmt.Errorf("timed out waiting for project release to stop")
	}
	return nil
}

func (project *Project) Delete() error {
	lock := projectLock(project.Name)
	lock.Lock()
	defer lock.Unlock()
	if err := project.Stop(); err != nil {
		return err
	}
	return project.Remove()
}

func (project *Project) Remove() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if strings.HasPrefix(project.State, "Error") {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "removing"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	allReleases := projectReleases(project.Name, true, 0)
	if !shutdownReleaseList(allReleases, 5*time.Second) {
		err := fmt.Errorf("timed out waiting for project releases to stop")
		project.State = fmt.Sprintf("Error: %v", err)
		if saveErr := project.Save(); saveErr != nil {
			return saveErr
		}
		notifyReleaseFailure(fmt.Sprintf("*%s* removal failed: %v", project.Name, err))
		return err
	}

	cmds := []string{
		fmt.Sprintf(`rm -f /etc/nginx/sites-available/%s`, project.Name),
		fmt.Sprintf(`rm -f /etc/nginx/sites-enabled/%s`, project.Name),
		fmt.Sprintf(`rm -rf %s`, project.ProjectDir()),
	}

	for _, cmd := range cmds {
		out, err := ExecWrapped(cmd)
		if err != nil {
			if LOG_MUTEX {
				slog.Info("Lock", "project", project.Name)
			}
			project.mu.Lock()
			project.State = fmt.Sprintf("Error: %v\n%s", err, out)
			project.mu.Unlock()
			if LOG_MUTEX {
				slog.Info("Unlock", "project", project.Name)
			}
			return project.Save()
		}
	}

	confMu.Lock()
	defer confMu.Unlock()
	conf, err := getConf()
	if err != nil {
		return err
	}
	newProjects := make([]Project, 0)
	for _, p := range conf.Projects {
		if p.Name != project.Name {
			newProjects = append(newProjects, p)
		}
	}
	conf.Projects = newProjects
	return setConf(conf)
}

func (project *Project) IsError() bool {
	return project.State != "" && strings.HasPrefix(project.State, "Error")
}

func (project *Project) Advance(force bool) error {
	lock := projectLock(project.Name)
	lock.Lock()
	defer lock.Unlock()
	return project.advance(force)
}

func (project *Project) advance(force bool) error {
	if force {
		if err := project.Clone(); err != nil {
			return err
		}
		return project.advance(false)
	}
	switch project.State {
	case "":
		if err := project.Clone(); err != nil {
			return err
		}
		return project.advance(false)
	case "cloning":
		return project.BuildAndRun()
	case "awaiting liveness", "🟢 running":
		return nil
	case "starting":
		if err := project.Proxy(); err != nil {
			return err
		}
		return project.advance(false)
	case "proxying":
		if err := project.Cert(); err != nil {
			return err
		}
		return project.advance(false)
	case "cleanup":
		return project.Cleanup()
	default:
		if project.IsError() {
			return nil
		}
		return fmt.Errorf("unknown project state %q", project.State)
	}
}

func GetProject(name string) (Project, error) {
	conf, err := GetConf()
	if err != nil {
		return Project{}, err
	}
	for _, project := range conf.Projects {
		if project.Name == name {
			return project, nil
		}
	}
	return Project{}, errors.New("project not found")
}

func IsAuthenticated(email, password string) (bool, error) {
	conf, err := GetConf()
	if err != nil {
		return false, err
	}
	if len(conf.Users) == 0 || password == "" || email == "" {
		return false, nil
	}

	for _, user := range conf.Users {
		if user.Email == email {
			hash := pbkdf2.Key([]byte(password), []byte(user.Salt), 10000, 512, sha512.New)
			return subtle.ConstantTimeCompare(user.Hash, hash) == 1, nil
		}
	}
	return false, nil
}

func SetUser(email, password string) error {
	confMu.Lock()
	defer confMu.Unlock()
	conf, err := getConf()
	if err != nil {
		return err
	}

	salt := make([]byte, 128)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	saltStr := base64.StdEncoding.EncodeToString(salt)
	hash := pbkdf2.Key([]byte(password), []byte(saltStr), 10000, 512, sha512.New)

	userIndex := -1
	for i, user := range conf.Users {
		if user.Email == email {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		conf.Users = append(conf.Users, User{Email: email, Hash: hash, Salt: saltStr})
	} else {
		conf.Users[userIndex] = User{Email: email, Hash: hash, Salt: saltStr}
	}

	return setConf(conf)
}
