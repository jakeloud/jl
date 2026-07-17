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
	"strings"
	"sync"

	"github.com/jakeloud/jl/ip_getter"
	"golang.org/x/crypto/pbkdf2"
)

const (
        PROJECTS_ROOT = "/app"
	JAKELOUD    = "jakeloud"
	LOG_MUTEX   = false
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
	Projects  []Project  `json:"apps"`
	Users []User `json:"users"`
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
			Projects:  []Project{{Name: JAKELOUD, Port: 666}},
			Users: []User{},
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
	project.State = "building"
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

func (project *Project) ShortRepoPath() (string, error) {
	parts := strings.Split(project.Repo, ":")
	if len(parts) < 2 {
		return "", errors.New("Repo format should be git@github.com:<user>/<repo>.git")
	}
	path := strings.Split(parts[1], ".git")[0]
	return path, nil
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

	repoPath, err := project.ShortRepoPath()
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

	_, err = ExecWrapped(fmt.Sprintf(`rm -rf %s/%s`, PROJECTS_ROOT, repoPath))
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

	cmd := fmt.Sprintf(`eval "$(ssh-agent -s)"; ssh-add %s; git clone --depth 1 %s %s/%s; kill $SSH_AGENT_PID`, SSH_KEY, project.Repo, PROJECTS_ROOT, repoPath)
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

	repoPath, err := project.ShortRepoPath()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`docker build -t %s %s/%s`, strings.ToLower(repoPath), PROJECTS_ROOT, repoPath)
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
	if project.State != "building" {
		return nil
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
	if project.State != "proxying" {
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

	repoPath, err := project.ShortRepoPath()
	if err != nil {
		return err
	}

	_, err = ExecWrapped(fmt.Sprintf(`if [ -z "$(docker ps -q -f name=%s)" ]; then echo "starting first time"; else docker stop %s && docker rm %s; fi`, project.Name, project.Name, project.Name))
	if err != nil {
		return err
	}

	dockerOptions := ""
	if opts, ok := project.Additional["dockerOptions"]; ok {
		dockerOptions = opts.(string)
	}

	cmd := fmt.Sprintf(`docker run --name %s -d -p %d:80 %s %s`, project.Name, project.Port, dockerOptions, strings.ToLower(repoPath))
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
	if project.State != "starting" || project.Domain == "" {
		return nil
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

	out, err := ExecWrapped(fmt.Sprintf(`docker stop %s`, project.Name))
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

func (project *Project) Remove(removeRepo bool) error {
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

	repoPath, err := project.ShortRepoPath()
	if err != nil {
		return err
	}

	cmds := []string{
		fmt.Sprintf(`docker rm %s`, project.Name),
		fmt.Sprintf(`rm -f /etc/nginx/sites-available/%s`, project.Name),
		fmt.Sprintf(`rm -f /etc/nginx/sites-enabled/%s`, project.Name),
	}
	if removeRepo {
		cmds = append(cmds, fmt.Sprintf(`docker image rm %s && rm -r %s/%s`, strings.ToLower(repoPath), PROJECTS_ROOT, repoPath))
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
		project.Build()
		break
	case "building":
		project.Proxy()
		break
	case "proxying":
		project.Start()
		break
	case "starting":
		project.Cert()
		break
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
