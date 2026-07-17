package api

import (
	"fmt"
	"github.com/jakeloud/jl/entities"
)

func GetProject(params apiRequest) (interface{}, error) {
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

	project, err := entities.GetProject(params.Name)
	if err != nil {
		return nil, err
	}
	if project.Additional == nil {
		project.Additional = make(map[string]interface{})
	}
	if project.State == "🟢 running" {
		containerName, err := project.CurrentContainerName()
		if err != nil {
			project.Additional["logs"] = fmt.Sprintf("Failed to get current container: %v", err)
			project.Additional["ps"] = fmt.Sprintf("Failed to get current container: %v", err)
			return project, nil
		}

		cmd := fmt.Sprintf("docker logs %s", containerName)
		out, err := entities.ExecWrapped(cmd)
		if err == nil {
			project.Additional["logs"] = out
		} else {
			project.Additional["logs"] = fmt.Sprintf("Failed to get logs: %v", err)
		}

		cmd = fmt.Sprintf("docker ps --format json -f name=%s", containerName)
		out, err = entities.ExecWrapped(cmd)
		if err == nil {
			project.Additional["ps"] = out
		} else {
			project.Additional["ps"] = fmt.Sprintf("Failed to get ps: %v", err)
		}
	}

	return project, nil
}
