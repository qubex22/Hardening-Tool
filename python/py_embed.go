//go:build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kluctl/go-embed-python/v2/embed"
)

func main() {
	var dumpAssets string
	flag.StringVar(&dumpAssets, "dump-assets", "", "Dump Python assets to directory")
	flag.Parse()

	if dumpAssets != "" {
		if err := dumpPythonAssets(dumpAssets); err != nil {
			log.Fatalf("Failed to dump Python assets: %v", err)
		}
		fmt.Printf("Python assets dumped to %s\n", dumpAssets)
		return
	}

	// Default behavior: initialize embedded Python
	initEmbeddedPython()
}

func dumpPythonAssets(outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// Embed Python distribution and extract to output directory
	e, err := embed.New(embed.WithPythonVersion("3.11"))
	if err != nil {
		return fmt.Errorf("failed to initialize embed: %w", err)
	}

	// Extract all files to output directory
	pythonDir := e.PythonDir()
	files, err := os.ReadDir(pythonDir)
	if err != nil {
		return fmt.Errorf("failed to read python dir: %w", err)
	}

	for _, f := range files {
		srcPath := filepath.Join(pythonDir, f.Name())
		dstPath := filepath.Join(outputDir, f.Name())

		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", srcPath, err)
		}

		if info.IsDir() {
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", dstPath, err)
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", srcPath, err)
			}
			if err := os.WriteFile(dstPath, data, info.Mode()); err != nil {
				return fmt.Errorf("failed to write %s: %w", dstPath, err)
			}
		}
	}

	return nil
}

func initEmbeddedPython() {
	// Initialize the embedded Python runtime
	e, err := embed.New(embed.WithPythonVersion("3.11"))
	if err != nil {
		log.Fatalf("Failed to initialize embedded Python: %v", err)
	}

	fmt.Printf("Embedded Python initialized at: %s\n", e.PythonDir())
	fmt.Println("Use py_embed.RunScript() to execute Python code.")
}
