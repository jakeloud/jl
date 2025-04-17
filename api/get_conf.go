package api

import (
	"github.com/jakeloud/jl/entities"
)

// GetConf retrieves the configuration based on user authentication.
func GetConf(params apiRequest) (interface{}, error) {
	conf, err := entities.GetConf()
	if err != nil {
		return nil, err
	}

	if len(conf.Users) == 0 {
		jakeloudApp, err := entities.GetApp(entities.JAKELOUD)
		if err != nil {
			return nil, err
		}
		if jakeloudApp.Email == "" {
			return map[string]string{"message": "domain"}, nil
		}
		return map[string]string{"message": "register"}, nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return nil, err
	}
	if !authenticated {
		return map[string]string{"message": "login"}, nil
	}

	return conf, nil
}
