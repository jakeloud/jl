package setup

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	srcFile, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %v", err)
	}
	defer srcFile.Close()

	destPath := filepath.Join("/etc/jakeloud", "jl")

	fmt.Printf("Copying %s to %s\n", exePath, destPath)
	if !dry {
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file %s: %v", destPath, err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy binary: %v", err)
		}

		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %v", err)
		}
	}

	return nil
}
