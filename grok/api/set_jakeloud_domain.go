package api

import (
	"github.com/jakeloud/jl/entities"
)

// SetJakeloudDomain updates the domain and email of the JAKELOUD app if authenticated.
func SetJakeloudDomain(params struct {
	Email    string
	Password string
	Domain   string
}) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

  if params.Email == "" {
    return nil
  }
  if len(conf.Users) > 0 {
    isAuth, err := entities.IsAuthenticated(params.Email, params.Password)
    if !isAuth || err != nil {
      return nil
    }
  }

	jakeloudApp, err := entities.GetApp(entities.JAKELOUD)
	if err != nil {
		return err
	}

	if params.Domain == "" {
		params.Domain = jakeloudApp.Domain
	}

	jakeloudApp.Domain = params.Domain
	jakeloudApp.Email = params.Email
	jakeloudApp.State = "building"

	if err := jakeloudApp.Save(); err != nil {
		return err
	}
	if err := jakeloudApp.Proxy(); err != nil {
		return err
	}
	if err := jakeloudApp.LoadState(); err != nil {
		return err
	}
	if jakeloudApp.Email != "" {
		jakeloudApp.State = "starting"
		if err := jakeloudApp.Save(); err != nil {
			return err
		}
		if err := jakeloudApp.Cert(); err != nil {
			return err
		}
	}

	return nil
}
