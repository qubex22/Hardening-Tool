package python

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed bundled
var bundledFS embed.FS

// InstallBundledAnsible installs ansible-core and ansible collections from bundled assets.
// No network access required.
func InstallBundledAnsible(pythonDir string) error {
	// Create a temp dir for extracted wheels
	tmpDir, err := os.MkdirTemp("", "harden_bundled_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract bundled files to temp dir (skip .placeholder files)
	if err := fs.WalkDir(bundledFS, "bundled", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".placeholder") || strings.HasSuffix(path, ".gitkeep") {
			return nil
		}
		content, err := bundledFS.ReadFile(path)
		if err != nil {
			return err
		}
		outPath := filepath.Join(tmpDir, filepath.Base(path))
		return os.WriteFile(outPath, content, 0644)
	}); err != nil {
		return fmt.Errorf("failed to extract bundled packages: %w", err)
	}

	pythonExe := filepath.Join(pythonDir, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(pythonDir, "Scripts", "python.exe")
	}

	// Install ansible-core from bundled wheels (no network)
	fmt.Println("Installing ansible-core from bundled packages...")
	installCmd := exec.Command(pythonExe, "-m", "pip", "install", "--quiet",
		"--no-index", "--find-links", tmpDir, "ansible-core")
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install ansible-core failed: %w: %s", err, string(output))
	}

	// Install ansible collections from bundled archives
	collectionDir := getAnsibleCollectionsDir(pythonDir)
	if err := installBundledCollections(pythonExe, tmpDir, collectionDir); err != nil {
		return err
	}

	fmt.Println("Ansible installed successfully.")
	return nil
}

// getAnsibleCollectionsDir returns the ansible_collections directory path
func getAnsibleCollectionsDir(pythonDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(pythonDir, "Lib", "ansible_collections")
	}
	return filepath.Join(pythonDir, "lib", "ansible_collections")
}

// installBundledCollections installs ansible collections from bundled tarballs
func installBundledCollections(pythonExe, tmpDir, collectionDir string) error {
	// List collection archives in tmpDir
	collectionFiles, err := filepath.Glob(filepath.Join(tmpDir, "ansible-*.tar.gz"))
	if err != nil {
		return err
	}

	for _, cf := range collectionFiles {
		collectionName := filepath.Base(cf)
		fmt.Printf("Installing collection: %s\n", collectionName)
		galaxyCmd := exec.Command(pythonExe, "-m", "ansible-galaxy", "collection", "install",
			cf, "--collections-path", collectionDir, "--force")
		if output, err := galaxyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ansible-galaxy failed for %s: %w: %s", collectionName, err, string(output))
		}
	}

	// Also handle zip files
	collectionFiles, err = filepath.Glob(filepath.Join(tmpDir, "ansible-*.zip"))
	if err != nil {
		return err
	}
	for _, cf := range collectionFiles {
		collectionName := filepath.Base(cf)
		fmt.Printf("Installing collection: %s\n", collectionName)
		galaxyCmd := exec.Command(pythonExe, "-m", "ansible-galaxy", "collection", "install",
			cf, "--collections-path", collectionDir, "--force")
		if output, err := galaxyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ansible-galaxy failed for %s: %w: %s", collectionName, err, string(output))
		}
	}

	return nil
}

