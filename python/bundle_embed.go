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
	// go:embed bundled embeds the *contents* of the bundled/ directory
	if err := fs.WalkDir(bundledFS, ".", func(path string, d fs.DirEntry, err error) error {
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

	var pythonExe string
	if runtime.GOOS == "windows" {
		pythonExe = filepath.Join(pythonDir, "Scripts", "python.exe")
	} else {
		pythonExe = filepath.Join(pythonDir, "bin", "python3")
		// Fallback to 'python' if 'python3' doesn't exist
		if _, err := os.Stat(pythonExe); err != nil {
			pythonExe = filepath.Join(pythonDir, "bin", "python")
		}
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
	// Try the top-level ansible_collections first (what ansible-galaxy creates)
	if runtime.GOOS == "windows" {
		candidates := []string{
			filepath.Join(pythonDir, "Lib", "ansible_collections"),
			filepath.Join(pythonDir, "Lib", "site-packages", "ansible_collections"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
		return filepath.Join(pythonDir, "Lib", "ansible_collections")
	}
	candidates := []string{
		filepath.Join(pythonDir, "lib", "ansible_collections"),
		filepath.Join(pythonDir, "lib", "python3", "site-packages", "ansible_collections"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return filepath.Join(pythonDir, "lib", "ansible_collections")
}

// installBundledCollections installs ansible collections from bundled tarballs
func installBundledCollections(pythonExe, tmpDir, collectionDir string) error {
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

// ensureAnsiblePosixCollection extracts and installs the bundled ansible.posix collection
func ensureAnsiblePosixCollection(pythonExe, pythonDir string) error {
	// Find the ansible.posix tarball in the embedded bundled files
	var collectionTarballPath string
	err := fs.WalkDir(bundledFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, "ansible-posix.tar.gz") {
			collectionTarballPath = path
			return fmt.Errorf("found")
		}
		return nil
	})

	// If no tarball found, collection may not be bundled — skip silently
	if collectionTarballPath == "" {
		fmt.Println("No bundled ansible.posix collection found, skipping.")
		return nil
	}

	// Extract the tarball content to a temp dir
	tmpDir, err := os.MkdirTemp("", "harden_collection_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Read the tarball content
	tarballContent, err := bundledFS.ReadFile(collectionTarballPath)
	if err != nil {
		return fmt.Errorf("failed to read bundled collection: %w", err)
	}

	// Write to temp file
	tarballPath := filepath.Join(tmpDir, "ansible-posix.tar.gz")
	if err := os.WriteFile(tarballPath, tarballContent, 0644); err != nil {
		return fmt.Errorf("failed to write temp tarball: %w", err)
	}

	// Ensure the collections directory exists
	collectionDir := getAnsibleCollectionsDir(pythonDir)
	if err := os.MkdirAll(collectionDir, 0755); err != nil {
		return fmt.Errorf("failed to create collections dir: %w", err)
	}

	// Install the collection using ansible-galaxy
	fmt.Println("Installing ansible.posix collection from bundled assets...")
	galaxyCmd := exec.Command(pythonExe, "-m", "ansible-galaxy", "collection", "install",
		tarballPath, "--collections-path", collectionDir, "--force")
	if output, err := galaxyCmd.CombinedOutput(); err != nil {
		// Check if it's already installed (idempotent)
		if strings.Contains(string(output), "already installed") || strings.Contains(string(output), "was installed successfully") {
			fmt.Println("ansible.posix collection already installed.")
			return nil
		}
		return fmt.Errorf("ansible-galaxy collection install failed: %w: %s", err, string(output))
	}

	fmt.Println("ansible.posix collection installed successfully.")
	return nil
}

