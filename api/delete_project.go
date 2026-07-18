package api

import (
	"github.com/jakeloud/jl/entities"
)

func DeleteProject(params apiRequest) error {
	project, err := entities.GetProject(params.Name)
	if err != nil {
		return nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return err
	}
	if !authenticated || params.Name == "" {
		return nil
	}

	return project.Delete()
}
