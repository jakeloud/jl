package setup

import (
	"fmt"
	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/ip_getter"
)

func link() (string, error) {
	app, err := entities.GetApp(entities.JAKELOUD)
	if err == nil {
		return fmt.Sprintf("https://%s", app.Domain), nil
	}

	l, err := ip_getter.GetPublicIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://jakeloud.%s.sslip.io", l), nil
}
