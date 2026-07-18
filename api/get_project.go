package api

import (
	"fmt"
	"os"

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
	currentRelease, err := project.CurrentReleaseNumber()
	if err == nil {
		logData, readErr := os.ReadFile(project.ReleaseLogPath(currentRelease))
		if readErr == nil {
			const maxLogSize = 64 * 1024
			if len(logData) > maxLogSize {
				logData = logData[len(logData)-maxLogSize:]
			}
			project.Additional["logs"] = string(logData)
		} else if os.IsNotExist(readErr) {
			project.Additional["logs"] = "No release log available"
		} else {
			project.Additional["logs"] = fmt.Sprintf("Failed to read logs: %v", readErr)
		}

		containerName := project.ReleaseContainerName(currentRelease)
		out, psErr := entities.ExecWrapped(fmt.Sprintf("docker ps --format json -f name=%s", containerName))
		if psErr == nil {
			project.Additional["ps"] = out
		} else {
			project.Additional["ps"] = fmt.Sprintf("Failed to get ps: %v", psErr)
		}
	} else {
		project.Additional["logs"] = fmt.Sprintf("Failed to get current release: %v", err)
	}

	return project, nil
}
