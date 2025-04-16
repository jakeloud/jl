package api

import (
	"github.com/jakeloud/jl/entities"
)

// SetJakeloudAdditional updates the additional field of the JAKELOUD app if authenticated and authorized.
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

	jakeloudApp, err := entities.GetApp(entities.JAKELOUD)
	if err != nil {
		return err
	}

	if params.Email != jakeloudApp.Email {
		return nil
	}

	jakeloudApp.Additional = params.Additional
	return jakeloudApp.Save()
}
