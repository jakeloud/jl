package api

import (
	"github.com/jakeloud/jl/entities"
)

// DeleteApp stops and removes an application if the user is authenticated.
func DeleteApp(params struct {
	Name     string
	Email    string
	Password string
}) error {
	app, err := entities.GetApp(params.Name)
	if err != nil {
		return nil // Silently return if app not found
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return err
	}
	if !authenticated || params.Name == "" {
		return nil
	}

	if err := app.Stop(); err != nil {
		return err
	}

	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	isRepoUsedElsewhere := false
	count := 0
	for _, a := range conf.Apps {
		if a.Repo == app.Repo {
			count++
			if count > 1 {
				isRepoUsedElsewhere = true
				break
			}
		}
	}

	return app.Remove(!isRepoUsedElsewhere)
}
