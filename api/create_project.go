package api

import (
	"fmt"

	"github.com/jakeloud/jl/entities"
)

func CreateProject(params apiRequest) error {

	// Validate authentication and required fields
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated || params.Repo == "" || params.Name == "" || params.Email == "" {
		return nil
	}

	commands := []string{}
	if params.Additional != nil {
		if value, exists := params.Additional["cmd"]; exists {
			values, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("additional.cmd must be an array of strings")
			}
			for _, value := range values {
				command, ok := value.(string)
				if !ok {
					return fmt.Errorf("additional.cmd must contain only strings")
				}
				commands = append(commands, command)
			}
		}
	}
	project := entities.Project{
		Email:      params.Email,
		Domain:     params.Domain,
		Repo:       params.Repo,
		Name:       params.Name,
		Additional: map[string]interface{}{"cmd": commands},
	}
	if _, err := project.ProjectDomains(); err != nil {
		return fmt.Errorf("invalid project domains: %v", err)
	}

	if err := project.DeployWithNewPort(); err != nil {
		return fmt.Errorf("failed to deploy project: %v", err)
	}

	if err := project.LoadState(); err != nil {
		return fmt.Errorf("failed to load project state: %v", err)
	}

	return nil
}
