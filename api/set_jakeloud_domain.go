package api

import (
	"github.com/jakeloud/jl/entities"

	"errors"
)

func SetJakeloudDomain(params apiRequest) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	if params.Email == "" {
		return errors.New("Email is required")
	}
	if len(conf.Users) > 0 {
		isAuth, err := entities.IsAuthenticated(params.Email, params.Password)
		if !isAuth || err != nil {
			return nil
		}
	}

	jakeloudProject, err := entities.GetProject(entities.JAKELOUD)
	if err != nil {
		return err
	}

	if params.Domain == "" {
		params.Domain = jakeloudProject.Domain
	}

	jakeloudProject.Domain = params.Domain
	jakeloudProject.Email = params.Email
	jakeloudProject.State = "starting"

	if err := jakeloudProject.Save(); err != nil {
		return err
	}
	if err := jakeloudProject.Proxy(); err != nil {
		return err
	}
	if err := jakeloudProject.LoadState(); err != nil {
		return err
	}
	if jakeloudProject.Email != "" {
		if err := jakeloudProject.Cert(); err != nil {
			return err
		}
	}

	return nil
}
