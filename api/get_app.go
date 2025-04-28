package api

import (
	"fmt"
	"github.com/jakeloud/jl/entities"
)

func GetApp(params apiRequest) (interface{}, error) {
	conf, err := entities.GetConf()
	if err != nil {
		return nil, err
	}
	if len(conf.Users) == 0 {
		return map[string]string{"message": "register"}, nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return nil, err
	}
	if !authenticated {
		return map[string]string{"message": "login"}, nil
	}

	if params.Name == "" {
		return map[string]string{"message": "name is undefined"}, nil
	}

	app, err := entities.GetApp(params.Name)
	if err != nil {
		return nil, err
	}
	if app.State == "🟢 running" {
		cmd := fmt.Sprintf("docker logs %s", params.Name)
		out, err := entities.ExecWrapped(cmd)
		if err == nil {
			app.Additional["logs"] = out
		} else {
			app.Additional["logs"] = fmt.Sprintf("Failed to get logs: %v", err)
		}

		cmd = fmt.Sprintf("docker ps --format json -f name=%s", params.Name)
		out, err = entities.ExecWrapped(cmd)
		if err == nil {
			app.Additional["ps"] = out
		} else {
			app.Additional["ps"] = fmt.Sprintf("Failed to get ps: %v", err)
		}
	}

	return app, nil
}
