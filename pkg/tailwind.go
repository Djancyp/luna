package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Tailwind CSS directives
const tailwindCSSInput = `
@tailwind base;
@tailwind components;
@tailwind utilities;
`

func Tailwind(baseDir string) (string, error) {
	// Check if npx is available
	if err := exec.Command("npx", "--version").Run(); err != nil {
		return "", fmt.Errorf("npx is not available: %v", err)
	}

	// Check if tailwind.config.js exists in the base directory
	configPath := filepath.Join(baseDir, "tailwind.config.js")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("tailwind.config.js not found at %s", configPath)
	}

	// Create a temporary directory for the virtual files
	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tempDir: %v", err)
	}
	// Paths for temporary files
	inputCSSPath := filepath.Join(tempDir, "input.css")
	outputCSSPath := filepath.Join(tempDir, "output.css")

	// Write virtual CSS input file
	if err := os.WriteFile(inputCSSPath, []byte(tailwindCSSInput), 0644); err != nil {
		return "", fmt.Errorf("failed to write input.css: %v", err)
	}

	// Use relative path for config so it's resolved from the working directory
	cmd := exec.Command("npx", "tailwindcss", "-i", inputCSSPath, "-o", outputCSSPath, "--config", "tailwind.config.js", "--minify")
	cmd.Dir = baseDir // Set the working directory to baseDir so relative paths work correctly

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	// Check if the command failed
	if err != nil {
		return "", fmt.Errorf("tailwindcss build failed: %v, output: %s", err, string(output))
	}

	// Read the generated output CSS
	outputCSS, err := os.ReadFile(outputCSSPath)
	if err != nil {
		return "", fmt.Errorf("failed to read output.css: %v", err)
	}
	return string(outputCSS), nil
}
