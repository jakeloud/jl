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
	JAKELOUD    = "jakeloud"
	CONF_FILE   = "/etc/jakeloud/conf.json"
	SSH_KEY     = "/etc/jakeloud/id_rsa"
	SSH_KEY_PUB = "/etc/jakeloud/id_rsa.pub"
	LOG_MUTEX   = false
)

var dry bool = false
var dry_conf []byte = []byte("{\"apps\":[{\"name\":\"jakeloud\",\"port\":666}],\"users\":[]}")

func SetDry(d bool) {
	dry = d
}

type App struct {
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
	Apps  []App  `json:"apps"`
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
			Apps:  []App{{Name: JAKELOUD, Port: 666}},
			Users: []User{},
		}
		return conf, nil
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

func execWrapped(cmd string) (string, error) {
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

	res, err := execWrapped("docker system prune -af")
	if err != nil {
		return err.Error(), err
	}
	return res, nil
}

func Start(server interface{}) error {
	app, err := GetApp(JAKELOUD)
	if err != nil {
		return err
	}

	if !dry {
		if _, err := os.Stat(SSH_KEY); os.IsNotExist(err) {
			_, err := execWrapped(`ssh-keygen -q -t ed25519 -N '' -f ` + SSH_KEY)
			if err != nil {
				return err
			}
		}

		sshKey, err := ioutil.ReadFile(SSH_KEY_PUB)
		if err != nil {
			return err
		}

		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		if app.Additional == nil {
			app.Additional = make(map[string]interface{})
		}
		app.Additional["sshKey"] = string(sshKey)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}

		if err := app.Save(); err != nil {
			return err
		}
	}

	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "building"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}

	if app.Domain == "" {
		dom, err := ip_getter.GetPublicIP()
		if err != nil {
			slog.Info("Failed to get ip", "err", err)
			return err
		}
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.Domain = fmt.Sprintf("jakeloud.%s.sslip.io", dom)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
	}

	if err := app.Save(); err != nil {
		return err
	}
	if err := app.Proxy(); err != nil {
		return err
	}
	if err := app.LoadState(); err != nil {
		return err
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "starting"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}
	if err := app.Cert(); err != nil {
		return err
	}

	return nil
}

func (app *App) Save() error {
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}

	conf, err := GetConf()
	if err != nil {
		return err
	}

	appIndex := -1
	for i, a := range conf.Apps {
		if a.Name == app.Name {
			appIndex = i
			break
		}
	}

	if appIndex == -1 {
		conf.Apps = append(conf.Apps, *app)
	} else {
		conf.Apps[appIndex] = *app
	}

	return SetConf(conf)
}

func (app *App) ShortRepoPath() (string, error) {
	parts := strings.Split(app.Repo, ":")
	if len(parts) < 2 {
		return "", errors.New("Repo format should be git@github.com:<user>/<repo>.git")
	}
	path := strings.Split(parts[1], ".git")[0]
	return path, nil
}

func (app *App) LoadState() error {
	loadedApp, err := GetApp(app.Name)
	if err != nil {
		return err
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = loadedApp.State
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	return nil
}

func (app *App) Clone() error {
	slog.Info("Cloning", "app", app.Name)
	if err := app.LoadState(); err != nil {
		return err
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "cloning"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	repoPath, err := app.ShortRepoPath()
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v", err)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}

	_, err = execWrapped(fmt.Sprintf(`rm -rf /etc/jakeloud/%s`, repoPath))
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v", err)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}

	cmd := fmt.Sprintf(`eval "$(ssh-agent -s)"; ssh-add %s; git clone --depth 1 %s /etc/jakeloud/%s; kill $SSH_AGENT_PID`, SSH_KEY, app.Repo, repoPath)
	_, err = execWrapped(cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v", err)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}
	return nil
}

func (app *App) Build() error {
	if err := app.LoadState(); err != nil {
		return err
	}
	if app.State != "cloning" {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "building"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	repoPath, err := app.ShortRepoPath()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf(`docker build -t %s /etc/jakeloud/%s`, strings.ToLower(repoPath), repoPath)
	out, err := execWrapped(cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}
	return nil
}

func (app *App) Proxy() error {
	if err := app.LoadState(); err != nil {
		return err
	}
	if app.State != "building" {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "proxying"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	server_name := "undefined"
	if app.Domain != "" {
		server_name = app.Domain
	}

	content := fmt.Sprintf(`
server {
	listen 80;
	server_name %s;

	location / {
		proxy_set_header   X-Forwarded-For $remote_addr;
		proxy_set_header   Host $host;
		proxy_pass         http://127.0.0.1:%d;

		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
	}
}`, server_name, app.Port)

	file := "default"
	if app.Name != JAKELOUD {
		file = app.Name
	}
	filePath := fmt.Sprintf("/etc/nginx/sites-available/%s", file)

	if dry {
		slog.Info("Writing", "file", filePath, "content", content)
	} else {
		if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	if out, err := execWrapped("nginx -t"); err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
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

	if out, err := execWrapped("service nginx restart"); err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}
	return nil
}

func (app *App) Start() error {
	if err := app.LoadState(); err != nil {
		return err
	}
	if app.State != "proxying" {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "starting"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	repoPath, err := app.ShortRepoPath()
	if err != nil {
		return err
	}

	_, err = execWrapped(fmt.Sprintf(`if [ -z "$(docker ps -q -f name=%s)" ]; then echo "starting first time"; else docker stop %s && docker rm %s; fi`, app.Name, app.Name, app.Name))
	if err != nil {
		return err
	}

	dockerOptions := ""
	if opts, ok := app.Additional["dockerOptions"]; ok {
		dockerOptions = opts.(string)
	}

	cmd := fmt.Sprintf(`docker run --name %s -d -p %d:80 %s %s`, app.Name, app.Port, dockerOptions, strings.ToLower(repoPath))
	out, err := execWrapped(cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}
	return nil
}

func (app *App) Cert() error {
	if err := app.LoadState(); err != nil {
		return err
	}
	if app.State != "starting" || app.Domain == "" {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "certing"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	email := app.Email
	if email == "" {
		email = "no-reply@gmail.com"
	}
	cmd := fmt.Sprintf(`certbot -n --agree-tos --email %s --nginx -d %s`, email, app.Domain)
	out, err := execWrapped(cmd)
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}

	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "🟢 running"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	return app.Save()
}

func (app *App) Stop() error {
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "stopping"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	out, err := execWrapped(fmt.Sprintf(`docker stop %s`, app.Name))
	if err != nil {
		if LOG_MUTEX {
			slog.Info("Lock", "app", app.Name)
		}
		app.mu.Lock()
		app.State = fmt.Sprintf("Error: %v\n%s", err, out)
		app.mu.Unlock()
		if LOG_MUTEX {
			slog.Info("Unlock", "app", app.Name)
		}
		return app.Save()
	}
	return nil
}

func (app *App) Remove(removeRepo bool) error {
	if err := app.LoadState(); err != nil {
		return err
	}
	if strings.HasPrefix(app.State, "Error") {
		return nil
	}
	if LOG_MUTEX {
		slog.Info("Lock", "app", app.Name)
	}
	app.mu.Lock()
	app.State = "removing"
	app.mu.Unlock()
	if LOG_MUTEX {
		slog.Info("Unlock", "app", app.Name)
	}
	if err := app.Save(); err != nil {
		return err
	}

	repoPath, err := app.ShortRepoPath()
	if err != nil {
		return err
	}

	cmds := []string{
		fmt.Sprintf(`docker rm %s`, app.Name),
		fmt.Sprintf(`rm -f /etc/nginx/sites-available/%s`, app.Name),
		fmt.Sprintf(`rm -f /etc/nginx/sites-enabled/%s`, app.Name),
	}
	if removeRepo {
		cmds = append(cmds, fmt.Sprintf(`docker image rm %s && rm -r /etc/jakeloud/%s`, strings.ToLower(repoPath), repoPath))
	}

	for _, cmd := range cmds {
		out, err := execWrapped(cmd)
		if err != nil {
			if LOG_MUTEX {
				slog.Info("Lock", "app", app.Name)
			}
			app.mu.Lock()
			app.State = fmt.Sprintf("Error: %v\n%s", err, out)
			app.mu.Unlock()
			if LOG_MUTEX {
				slog.Info("Unlock", "app", app.Name)
			}
			return app.Save()
		}
	}

	conf, err := GetConf()
	if err != nil {
		return err
	}
	newApps := make([]App, 0)
	for _, a := range conf.Apps {
		if a.Name != app.Name {
			newApps = append(newApps, a)
		}
	}
	conf.Apps = newApps
	return SetConf(conf)
}

func (app *App) IsError() bool {
	return app.State != "" && strings.HasPrefix(app.State, "Error")
}

func (app *App) Advance(force bool) error {
	if (app.State == "🟢 running" || app.IsError()) && !force {
		return nil
	}
	switch app.State {
	case "cloning":
		app.Build()
		break
	case "building":
		app.Proxy()
		break
	case "proxying":
		app.Start()
		break
	case "starting":
		app.Cert()
		break
	default:
		if err := app.Clone(); err != nil {
			return err
		}
	}
	return app.Advance(false)
}

func GetApp(name string) (App, error) {
	conf, err := GetConf()
	if err != nil {
		return App{}, err
	}
	for _, app := range conf.Apps {
		if app.Name == name {
			return app, nil
		}
	}
	return App{}, errors.New("app not found")
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
