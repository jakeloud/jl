package setup

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed jakeloud.service
var embeddedFile embed.FS

const serviceDir = "/etc/systemd/system"

func setupService(dry bool) error {
	fmt.Printf("Creating dir %s\n", "/etc/jakeloud")
	if !dry {
		if err := os.MkdirAll("/etc/jakeloud", 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", "/etc/jakeloud", err)
		}
	}

	data, err := embeddedFile.ReadFile("jakeloud.service")
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %v", err)
	}

	outputPath := filepath.Join(serviceDir, "jakeloud.service")

	fmt.Printf("Writing %s\n", outputPath)
	if !dry {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", outputPath, err)
		}
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	destPath := filepath.Join("/usr/local/bin", "jl")
	rel, _ := filepath.Rel(exePath, destPath)
	shouldCopy := rel != "."

	if shouldCopy {
		fmt.Printf("Copying %s to %s\n", exePath, destPath)

		srcFile, err := os.Open(exePath)
		if err != nil {
			return fmt.Errorf("failed to open executable: %v\n", err)
		}
		defer srcFile.Close()
		if !dry {
			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				if err.Error() == "open /usr/local/bin/jl: text file busy" {
					return fmt.Errorf("Please stop jl with `systemctl stop jakeloud` before updating\n")
				}
				return fmt.Errorf("failed to create destination file %s: %v\n", destPath, err)
			}
			defer destFile.Close()

			if _, err := io.Copy(destFile, srcFile); err != nil {
				return fmt.Errorf("failed to copy binary: %v\n", err)
			}

			if err := os.Chmod(destPath, 0755); err != nil {
				return fmt.Errorf("failed to set executable permissions: %v\n", err)
			}
		}
	}

	fmt.Printf("Starting services\n")
	out, err := execWrapped(dry, "systemctl daemon-reload && systemctl enable jakeloud")
	if err != nil {
		if strings.HasPrefix(out, "System has not been booted with systemd as init system (PID 1)") {
			return fmt.Errorf("Systemctl has not been booted. Please, reboot your machine to enable it.\n")
		}
		return fmt.Errorf("failed to enable services: %v\n%s\n", err, out)
	}
	out, err = execWrapped(dry, "systemctl start jakeloud")
	if err != nil {
		return fmt.Errorf("failed to enable services: %v\n%s\n", err, out)
	}
	out, err = execWrapped(dry, "service nginx start && service nginx restart")
	if err != nil {
		return fmt.Errorf("failed to enable services: %v\n%s\n", err, out)
	}

	return nil
}
