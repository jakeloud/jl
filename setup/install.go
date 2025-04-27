package setup

import (
	"fmt"
)

func install(dry bool) {
	out, err := execWrapped(dry, "apt-get update && apt-get install -y systemd nginx certbot python3-certbot-nginx openssh-client git")
	if err != nil {
		fmt.Println(out)
		fmt.Print(err)
	}
}
