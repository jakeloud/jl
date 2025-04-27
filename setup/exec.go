package setup

import (
	"log/slog"
	"os/exec"
)

func execWrapped(dry bool, cmd string) (string, error) {
	if dry {
		slog.Info("Executing", "cmd", cmd)
		return "", nil
	}

	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
