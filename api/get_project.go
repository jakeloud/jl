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
		project.Additional["currentRelease"] = currentRelease
		runtime := entities.ReleaseRuntimeStatus(project.Name, currentRelease)
		project.Additional["runtime"] = runtime
		if runtime.PromotionDeadline != "" {
			project.Additional["promotionDeadline"] = runtime.PromotionDeadline
		}
		if runtime.Alive {
			status := "running"
			if runtime.Active {
				status = "active"
			}
			project.Additional["ps"] = fmt.Sprintf("%s (pid %d)", status, runtime.PID)
		} else {
			project.Additional["ps"] = "not running"
		}
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
	} else {
		project.Additional["logs"] = fmt.Sprintf("Failed to get current release: %v", err)
	}

	return project, nil
}
