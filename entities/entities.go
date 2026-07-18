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

var (
	confMu         sync.Mutex
	projectLocksMu sync.Mutex
	projectLocks   = make(map[string]*sync.Mutex)
)

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

	temp, err := os.CreateTemp(filepath.Dir(CONF_FILE), ".conf-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0644); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, CONF_FILE)
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

func (project *Project) DeployWithNewPort() error {
	lock := projectLock(project.Name)
	lock.Lock()
	defer lock.Unlock()

	confMu.Lock()
	conf, err := getConf()
	if err == nil {
		takenPorts := reservedReleasePorts()
		for _, configuredProject := range conf.Projects {
			takenPorts[configuredProject.Port] = true
		}
		project.Port = 38000
		for takenPorts[project.Port] {
			project.Port++
		}
		projectIndex := -1
		for i, configuredProject := range conf.Projects {
			if configuredProject.Name == project.Name {
				projectIndex = i
				break
			}
		}
		if projectIndex < 0 {
			conf.Projects = append(conf.Projects, *project)
		} else {
			conf.Projects[projectIndex] = *project
		}
		err = setConf(conf)
	}
	confMu.Unlock()
	if err != nil {
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

func (project *Project) Proxy() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "starting" {
		return nil
	}
	domain, _, err := ParseProjectDomain(project.Domain)
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
	if err := project.configureProxy(domain, project.Port); err != nil {
		project.State = fmt.Sprintf("Error: %v", err)
		return project.Save()
	}
	return nil
}

func (project *Project) configureProxy(domain string, port int) error {
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
}`, domain, port)

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
		return fmt.Errorf("nginx config test failed: %w\n%s", err, out)
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
		return fmt.Errorf("nginx restart failed: %w\n%s", err, out)
	}
	return nil
}

func (project *Project) Cert() error {
	if err := project.LoadState(); err != nil {
		return err
	}
	if project.State != "proxying" {
		return nil
	}
	domain, _, err := ParseProjectDomain(project.Domain)
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
		if shuttingDown.Load() {
			return errors.New("jakeloud is shutting down")
		}
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
