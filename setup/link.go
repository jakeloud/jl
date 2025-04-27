package setup

import (
	"fmt"
	"github.com/jakeloud/jl/ip_getter"
)

func link() (string, error) {
	l, err := ip_getter.GetPublicIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://jakeloud.%s.sslip.io", l), nil
}
