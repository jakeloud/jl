package setup

import (
	"fmt"
	"os"
	"time"
)

var dry bool = false

func Start(d bool) {
	dry = d
	start := time.Now()

	fmt.Printf("Installing required packages\n")
	install(dry)
	err := setupService(dry)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	cmd := "ssh-keyscan -H github.com >> ~/.ssh/known_hosts"
	_, err = execWrapped(dry, cmd)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	l, err := link()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	fmt.Printf("JakeLoud successfully installed! 🎊 took %s\n", time.Since(start))
	fmt.Printf("go to %s to finish installation\n", l)
}
