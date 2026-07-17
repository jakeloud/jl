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
	if !authenticated || params.Domain == "" || params.Repo == "" || params.Name == "" || params.Email == "" {
		return nil
	}

	conf, err := entities.GetConf()
	if err != nil {
		return fmt.Errorf("failed to get config: %v", err)
	}

	port := 0
	// Find an available port
	takenPorts := make(map[int]int)
	for _, p := range conf.Projects {
		n, exists := takenPorts[p.Port]
		if exists {
			takenPorts[p.Port] = n + 1
		} else {
			takenPorts[p.Port] = 1
		}
		if p.Name == params.Name {
			port = p.Port
		}
	}

	if port != 0 {
		n, _ := takenPorts[port]
		if n != 1 {
			port = 38000
			for takenPorts[port] > 0 {
				port++
			}
		}
	} else {
		port = 38000
		for takenPorts[port] > 0 {
			port++
		}
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
		Port:       port,
		Additional: map[string]interface{}{"dockerOptions": dockerOptions},
	}

	if err := project.Save(); err != nil {
		return fmt.Errorf("failed to save project: %v", err)
	}

	if err := project.Advance(true); err != nil {
		return fmt.Errorf("failed to advance project: %v", err)
	}

	dt := int(time.Since(startTime).Seconds())

	if err := project.LoadState(); err != nil {
		return fmt.Errorf("failed to load project state: %v", err)
	}

	logMessage := fmt.Sprintf("*%s* started\\. _%ds_", project.Name, dt)
	if project.IsError() {
		logMessage = fmt.Sprintf("*%s* Failed to start\\. _%ds_", project.Name, dt)
	}

	if err := logger.Log(logMessage); err != nil {
		return fmt.Errorf("failed to log project status: %v", err)
	}

	return nil
}
