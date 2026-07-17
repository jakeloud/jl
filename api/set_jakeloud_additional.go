package api

import (
	"github.com/jakeloud/jl/entities"
)

func SetJakeloudAdditional(params apiRequest) error {
	if params.Additional == nil {
		return nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return err
	}
	if !authenticated {
		return nil
	}

	jakeloudProject, err := entities.GetProject(entities.JAKELOUD)
	if err != nil {
		return err
	}

	if params.Email != jakeloudProject.Email {
		return nil
	}

	jakeloudProject.Additional = params.Additional
	return jakeloudProject.Save()
}
