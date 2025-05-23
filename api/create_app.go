package api

import (
	"fmt"
	"time"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
)

// CreateApp creates and starts a new application with the provided parameters.
func CreateApp(params apiRequest) error {
	startTime := time.Now()

	// Validate authentication and required fields
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated || params.Domain == "" || params.Repo == "" || params.Name == "" || params.Email == "" {
		return nil // Silently return as per original logic
	}

	// Get configuration
	conf, err := entities.GetConf()
	if err != nil {
		return fmt.Errorf("failed to get config: %v", err)
	}

	port := 0
	// Find an available port
	takenPorts := make(map[int]int)
	for _, app := range conf.Apps {
		n, exists := takenPorts[app.Port]
		if exists {
			takenPorts[app.Port] = n + 1
		} else {
			takenPorts[app.Port] = 1
		}
		if app.Name == params.Name {
			port = app.Port
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

	// Create new App instance
	app := entities.App{
		Email:      params.Email,
		Domain:     params.Domain,
		Repo:       params.Repo,
		Name:       params.Name,
		Port:       port,
		Additional: map[string]interface{}{"dockerOptions": dockerOptions},
	}

	// Save the app
	if err := app.Save(); err != nil {
		return fmt.Errorf("failed to save app: %v", err)
	}

	// Advance the app
	if err := app.Advance(true); err != nil {
		return fmt.Errorf("failed to advance app: %v", err)
	}

	// Calculate duration
	dt := int(time.Since(startTime).Seconds())

	// Load final state and log result
	if err := app.LoadState(); err != nil {
		return fmt.Errorf("failed to load app state: %v", err)
	}

	logMessage := fmt.Sprintf("*%s* started\\. _%ds_", app.Name, dt)
	if app.IsError() {
		logMessage = fmt.Sprintf("*%s* Failed to start\\. _%ds_", app.Name, dt)
	}

	if err := logger.Log(logMessage); err != nil {
		return fmt.Errorf("failed to log app status: %v", err)
	}

	return nil
}
