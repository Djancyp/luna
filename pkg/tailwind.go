package pkg

import (
	"log"
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

func Tailwind(baseDir string) string {
	// Create a temporary directory for the virtual files
	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Failed to create tempDir: %v", err)
	}
	// Paths for temporary files
	inputCSSPath := filepath.Join(tempDir, "input.css")
	config := filepath.Join("tailwind.config.js")
	outputCSSPath := filepath.Join(tempDir, "output.css")

	// Write virtual CSS input file
	if err := os.WriteFile(inputCSSPath, []byte(tailwindCSSInput), 0644); err != nil {
		log.Fatalf("Failed to write input.css: %v", err)
	}
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Run Tailwind CSS using npx, specifying input and output paths
	cmd := exec.Command("npx", "tailwindcss", "-i", inputCSSPath, "-o", outputCSSPath, "--config", config, "--minify")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = currentDir + "/" + baseDir

	// Execute the command
	if err := cmd.Run(); err != nil {
		log.Fatalf("Tailwind CSS build failed: %v", err)
	}

	// Read and print the generated output CSS
	outputCSS, err := os.ReadFile(outputCSSPath)
	if err != nil {
		log.Fatalf("Failed to read output.css: %v", err)
	}
	return string(outputCSS)
}
