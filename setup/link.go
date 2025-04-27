package setup

import (
	"fmt"
	"github.com/jakeloud/jl/ip_getter"
)

func link() {
	l, err := ip_getter.GetPublicIP()
	if err != nil {
		fmt.Println("failed to get ip of your server")
		return
	}
	domain := fmt.Sprintf("jakeloud.%s.sslip.io", l)
	fmt.Println("JakeLoud successfully installed! 🎊")
	fmt.Printf("visit https://%s to finish setup\n", domain)
}
