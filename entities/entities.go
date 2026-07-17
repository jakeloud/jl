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

	return nil
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

	cmd := fmt.Sprintf(`eval "$(ssh-agent -s)"; ssh-add %s; git clone --depth 1 %s %s; kill $SSH_AGENT_PID`, SSH_KEY, project.Repo, releaseDir)
	_, err = ExecWrapped(cmd)
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
	if opts, ok := project.Additional["dockerOptions"]; ok {
		dockerOptions = opts.(string)
	}

	cmd := fmt.Sprintf(`docker run --name %s -d -p %d:80 %s %s`, containerName, project.Port, dockerOptions, project.DockerImage())
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
	return nil
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

	currentContainer, err := project.CurrentContainerName()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`for container in $(docker ps -a --format '{{.Names}}' --filter name=^/%s-r); do if [ "$container" != "%s" ]; then docker stop "$container" || true; docker rm "$container" || true; fi; done`, project.Name, currentContainer)
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

	containerName, err := project.CurrentContainerName()
	if err != nil {
		return err
	}

	out, err := ExecWrapped(fmt.Sprintf(`docker stop %s`, containerName))
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

	cmds := []string{
		fmt.Sprintf(`for container in $(docker ps -a --format '{{.Names}}' --filter name=^/%s-r); do docker stop "$container" || true; docker rm "$container" || true; done`, project.Name),
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
