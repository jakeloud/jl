package api

import (
	"github.com/jakeloud/jl/entities"
)

func Register(params apiRequest) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	jakeloudProject, err := entities.GetProject(entities.JAKELOUD)
	if err != nil {
		return err
	}

	allowRegister, _ := jakeloudProject.Additional["allowRegister"].(bool)
	if !(allowRegister || len(conf.Users) == 0) || params.Email == "" || params.Password == "" {
		return nil
	}

	if len(conf.Users) == 0 {
		jakeloudProject.Email = params.Email
		if err := jakeloudProject.Save(); err != nil {
			return err
		}
	}

	return entities.SetUser(params.Email, params.Password)
}
