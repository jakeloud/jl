package api

import (
	"github.com/jakeloud/jl/entities"
)

// Register creates a new user if registration is allowed and inputs are valid.
func Register(params apiRequest) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	jakeloudApp, err := entities.GetApp(entities.JAKELOUD)
	if err != nil {
		return err
	}

	allowRegister, _ := jakeloudApp.Additional["allowRegister"].(bool)
	if !(allowRegister || len(conf.Users) == 0) || params.Email == "" || params.Password == "" {
		return nil
	}

	if len(conf.Users) == 0 {
		jakeloudApp.Email = params.Email
		if err := jakeloudApp.Save(); err != nil {
			return err
		}
	}

	return entities.SetUser(params.Email, params.Password)
}
