package api

import (
	"fmt"
	"time"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
)

// CreateApp creates and starts a new application with the provided parameters.
func CreateApp(params struct {
	Domain        string
	Repo          string
	Name          string
	DockerOptions string
	Password      string
	Email         string
}) error {
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

	// Find an available port
	takenPorts := make(map[int]bool)
	for _, app := range conf.Apps {
		takenPorts[app.Port] = true
	}
	port := 38000
	for takenPorts[port] {
		port++
	}

	// Create new App instance
	app := entities.App{
		Email:      params.Email,
		Domain:     params.Domain,
		Repo:       params.Repo,
		Name:       params.Name,
		Port:       port,
		Additional: map[string]interface{}{"dockerOptions": params.DockerOptions},
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

	if err := telegram.Log(logMessage); err != nil {
		return fmt.Errorf("failed to log app status: %v", err)
	}

	return nil
}
