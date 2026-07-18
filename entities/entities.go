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
	stopRequested atomic.Bool
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

func requestReleaseStop(release *Release) {
	if release.stopRequested.CompareAndSwap(false, true) && release.Cmd.Process != nil {
		if err := release.Cmd.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
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

	conf, err := GetConf()
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

	return SetConf(conf)
}

func (project *Project) ProjectDir() string {
	return filepath.Join(PROJECTS_ROOT, project.Name)
}

func (project *Project) DockerImage() string {
	return strings.ToLower(project.Name)
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

func (project *Project) Build() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "cloning" {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "project", project.Name)
	}
	project.mu.Lock()
	project.State = "building"
	project.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "project", project.Name)
	}
	if err := project.Save(); err != nil {
		return err
	}

	releaseDir, err := project.CurrentReleaseDir()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`docker build -t %s %s`, project.DockerImage(), releaseDir)
	releaseNumber, err := project.CurrentReleaseNumber()
	if err != nil {
		return err
	}
	out, err := project.runReleaseCommand(releaseNumber, cmd)
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

func (project *Project) Proxy() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "starting" {
		return nil
	}
	if project.Domain == "" {
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

	server_name := "undefined"
	if project.Domain != "" {
		server_name = project.Domain
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
}`, server_name, project.Port)

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

func (project *Project) Start() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "building" {
		return nil
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
	if err := project.Save(); err != nil {
		return err
	}

	containerName, err := project.CurrentContainerName()
	if err != nil {
		return err
	}

	dockerOptions := ""
	if opts, ok := project.Additional["dockerOptions"].(string); ok {
		dockerOptions = opts
	}

	releaseNumber, err := project.CurrentReleaseNumber()
	if err != nil {
		return err
	}
	logFile, err := os.OpenFile(project.ReleaseLogPath(releaseNumber), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(logFile, "\n--- %s ---\n$ docker run --rm --sig-proxy=true --name %s -p %d:80 %s %s\n", time.Now().Format(time.RFC3339), containerName, project.Port, dockerOptions, project.DockerImage())
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`exec docker run --rm --sig-proxy=true --name %s -p %d:80 %s %s`, containerName, project.Port, dockerOptions, project.DockerImage()))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
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

	release := &Release{
		ProjectName:   project.Name,
		Number:        releaseNumber,
		ContainerName: containerName,
		Cmd:           cmd,
		Done:          make(chan struct{}),
	}
	registerRelease(release)
	go waitForRelease(release, logFile)
	return nil
}

func waitForRelease(release *Release, logFile *os.File) {
	err := release.Cmd.Wait()
	_ = logFile.Close()
	unregisterRelease(release)
	close(release.Done)

	if err == nil || release.stopRequested.Load() || shuttingDown.Load() {
		return
	}

	project, getErr := GetProject(release.ProjectName)
	if getErr != nil {
		return
	}
	current, currentErr := project.CurrentReleaseNumber()
	if currentErr != nil || current != release.Number {
		return
	}
	project.State = fmt.Sprintf("Error: release r%d exited: %v", release.Number, err)
	if saveErr := project.Save(); saveErr != nil {
		slog.Info("Failed to save release failure", "project", release.ProjectName, "err", saveErr)
	}
	notifyReleaseFailure(fmt.Sprintf("*%s* release r%d failed: %v", release.ProjectName, release.Number, err))
}

func (project *Project) Cert() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "proxying" {
		return nil
	}
	if project.Domain == "" {
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
	cmd := fmt.Sprintf(`certbot -n --agree-tos --email %s --nginx -d %s`, email, project.Domain)
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

	conf, err := GetConf()
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
	return SetConf(conf)
}

func (project *Project) IsError() bool {
	return project.State != "" && strings.HasPrefix(project.State, "Error")
}

func (project *Project) Advance(force bool) error {
	if (project.State == "🟢 running" || project.IsError()) && !force {
		return nil
	}
	switch project.State {
	case "cloning":
		if err := project.Build(); err != nil {
			return err
		}
	case "building":
		if err := project.Start(); err != nil {
			return err
		}
	case "starting":
		if err := project.Proxy(); err != nil {
			return err
		}
	case "proxying":
		if err := project.Cert(); err != nil {
			return err
		}
	case "cleanup":
		if err := project.Cleanup(); err != nil {
			return err
		}
	default:
		if err := project.Clone(); err != nil {
			return err
		}
	}
	return project.Advance(false)
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
	conf, err := GetConf()
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

	return SetConf(conf)
}
