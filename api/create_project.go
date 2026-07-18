package api

import (
	"fmt"
	"time"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
)

func CreateProject(params apiRequest) error {
	startTime := time.Now()

	// Validate authentication and required fields
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated || params.Repo == "" || params.Name == "" || params.Email == "" {
		return nil
	}

	if _, _, err := entities.ParseProjectDomain(params.Domain); err != nil {
		return fmt.Errorf("invalid project domain: %v", err)
	}

	dockerOptions := params.DockerOptions
	if params.Additional != nil {
		tmp, exists := params.Additional["dockerOptions"].(string)
		if exists {
			dockerOptions = tmp
		}
	}

	project := entities.Project{
		Email:      params.Email,
		Domain:     params.Domain,
		Repo:       params.Repo,
		Name:       params.Name,
		Additional: map[string]interface{}{"dockerOptions": dockerOptions},
	}

	if err := project.DeployWithNewPort(); err != nil {
		return fmt.Errorf("failed to deploy project: %v", err)
	}

	dt := int(time.Since(startTime).Seconds())

	if err := project.LoadState(); err != nil {
		return fmt.Errorf("failed to load project state: %v", err)
	}

	logMessage := fmt.Sprintf("*%s* deployment started\\. _%ds_", project.Name, dt)
	if project.IsError() {
		logMessage = fmt.Sprintf("*%s* Failed to start\\. _%ds_", project.Name, dt)
	}

	if err := logger.Log(logMessage); err != nil {
		return fmt.Errorf("failed to log project status: %v", err)
	}

	return nil
}
