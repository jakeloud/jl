package api

import (
	"github.com/jakeloud/jl/entities"
)

// Register creates a new user if registration is allowed and inputs are valid.
func Register(params struct {
	Password string
	Email    string
}) error {
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

	return entities.SetUser(params.Email, params.Password)
}
