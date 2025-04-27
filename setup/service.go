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

const targetDir = "/etc/jakeloud"

func setupService(dry bool) error {
	// Create target directory
	fmt.Printf("Creating dir %s\n", targetDir)
	if !dry {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", targetDir, err)
		}
	}

	// Read embedded file
	data, err := embeddedFile.ReadFile("jakeloud.service")
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %v", err)
	}

	// Write embedded file to target directory
	outputPath := filepath.Join(targetDir, "jakeloud.service")

	fmt.Printf("Writing %s\n", outputPath)
	if !dry {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", outputPath, err)
		}
	}

	// Get path of current executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Open source binary
	srcFile, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %v", err)
	}
	defer srcFile.Close()

	// Create destination path for binary
	destPath := filepath.Join(targetDir, "jl")

	fmt.Printf("Copying %s to %s\n", exePath, destPath)
	if !dry {
		// Create destination file
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file %s: %v", destPath, err)
		}
		defer destFile.Close()

		// Copy binary
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy binary: %v", err)
		}

		// Make copied binary executable
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %v", err)
		}
	}

	return nil
}
