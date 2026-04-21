# Plan: Bundle ansible-core and ansible.posix for air-gapped deployment

## Overview
Bundle all Python dependencies (ansible-core, ansible.posix collection) into the binary at build time. The target host requires zero internet access.

## Files to create/modify

### 1. NEW: `python/pip_gen/main.go` — Pip package bundler
```go
package main

import (
	"fmt"
	"os"

	"github.com/kluctl/go-embed-python/pip"
)

func main() {
	targetDir := "./python/bundled"
	fmt.Println("Bundling ansible-core and ansible.posix collection...")
	err := pip.CreateEmbeddedPipPackagesForKnownPlatforms("requirements.txt", targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Bundling complete. Files written to:", targetDir)
}
```

### 2. NEW: `python/pip_gen/requirements.txt`
```
ansible-core
```

### 3. NEW: `python/pip_gen/go.mod`
```
module harden_sles15/python/pip_gen

go 1.21

require github.com/kluctl/go-embed-python v0.0.0-3.11.8-20240224-1
```

### 4. NEW: `python/bundled.go` — Embeds and extracts bundled packages
```go
package python

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed bundled
var bundledFS embed.FS

// ExtractBundledPackages extracts bundled pip packages (ansible-core, ansible.posix)
// to the given python directory, adding them to the site-packages and collections paths.
func ExtractBundledPackages(pythonDir string) error {
	return bundledFS.WalkDir("bundled", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, err := bundledFS.ReadFile(path)
		if err != nil {
			return err
		}
		// Determine target path based on file type
		var targetDir string
		if filepath.Ext(path) == ".whl" || filepath.Ext(path) == ".tar.gz" {
			// Wheel/source packages go into site-packages via pip install later
			// For now, we install them directly
			targetDir = filepath.Join(pythonDir, "lib")
		} else {
			targetDir = filepath.Dir(filepath.Join(pythonDir, path))
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(targetDir, filepath.Base(path)), content, 0644)
	})
}
```

Actually, the `go-embed-python` library provides a better approach. Let me reconsider...

### Revised approach using go-embed-python's embed_util

The library provides `embed_util.NewEmbeddedFiles()` which can extract embedded data. The flow is:

1. **Build step**: Run `go generate ./...` which executes `pip_gen/main.go` to create wheel files in `python/bundled/`
2. **Embed step**: Use `//go:embed python/bundled` to embed the wheels into the binary
3. **Runtime step**: Extract wheels to temp dir, then use embedded Python's pip to install them from the extracted location

### Simplified plan:

#### Step A: Create `python/pip_gen/` with generator that creates wheels
- Runs at build time with internet access
- Downloads ansible-core wheel + ansible.posix collection
- Saves as `.whl` files in `python/bundled/`

#### Step B: Embed wheels via `//go:embed python/bundled` in a new `python/bundle_embed.go`

#### Step C: At runtime in `ensureAnsible()`:
1. Extract wheels to a temp directory
2. Run `pip install /path/to/extracted/wheels/*` (no network needed)
3. Run `ansible-galaxy collection install` from the extracted ansible.posix tarball

### 5. MODIFY: `python/runtime.go` — `ensureAnsible()` function
Replace the network-dependent code with:
```go
func (r *PythonRuntime) ensureAnsible() error {
	pythonExe := r.ep.GetExePath()
	if _, err := os.Stat(pythonExe); err != nil {
		return fmt.Errorf("embedded Python not found at %s", pythonExe)
	}

	// Check if ansible is already installed (from bundled wheels)
	checkCmd := exec.Command(pythonExe, "-c", "import ansible; print(ansible.__version__)")
	if output, err := checkCmd.CombinedOutput(); err == nil {
		fmt.Printf("Ansible %s already installed.\n", strings.TrimSpace(string(output)))
		return nil
	}

	// Install from bundled wheels (no network needed)
	fmt.Println("Installing ansible from bundled packages...")
	bundledWheelsDir := ExtractBundledPackagesToTemp()  // extracts + returns path

	installCmd := exec.Command(pythonExe, "-m", "pip", "install", "--quiet", "--no-index", "--find-links", bundledWheelsDir, "ansible-core")
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install ansible-core failed: %w: %s", err, string(output))
	}

	// Install ansible.posix collection from bundled tarball
	fmt.Println("Installing ansible.posix collection...")
	collectionDir := filepath.Join(r.ep.GetExtractedPath(), "lib", "ansible_collections")
	extractedCollection := ExtractAnsiblePosixCollection()  // extracts to temp
	galaxyCmd := exec.Command(pythonExe, "-m", "ansible-galaxy", "collection", "install", extractedCollection, "--collections-path", collectionDir, "--force")
	if output, err := galaxyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ansible-galaxy failed: %w: %s", err, string(output))
	}

	fmt.Println("Ansible installed successfully.")
	return nil
}
```

### 6. MODIFY: `build.sh`
Add step before `go build`:
```bash
# Step 2: Generate bundled pip packages (needs internet)
echo ""
echo "[2/6] Generating bundled pip packages..."
cd python/pip_gen && go run main.go && cd ../..
echo "Bundled packages generated."

# Step 3: Embed bundled packages
echo ""
echo "[3/6] Embedding bundled packages..."
go generate ./...
echo "Bundled packages embedded."
```

### 7. MODIFY: `Dockerfile.build`
Ensure internet is available during build (it already is in the build container).

### 8. MODIFY: `.gitignore`
Add `python/bundled/` (generated, large files).

## File summary

**New files:**
- `python/pip_gen/main.go` — generator that downloads wheels
- `python/pip_gen/requirements.txt` — lists ansible-core
- `python/pip_gen/go.mod` — go module for the generator
- `python/bundle_embed.go` — embeds wheels + provides extraction functions

**Modified files:**
- `python/runtime.go` — `ensureAnsible()` uses bundled wheels instead of network
- `build.sh` — runs generator before compilation
- `.gitignore` — ignore `python/bundled/`

**Not modified:**
- `ansible/` — playbook and roles stay the same
- `main.go` — no changes needed
- `ansible_runner/` — no changes needed

## Build flow (with internet at build time)
1. `pip_gen` downloads ansible-core wheel + ansible.posix collection
2. Wheels are embedded into the binary via `//go:embed`
3. `go build` compiles the binary

## Runtime flow (air-gapped, no internet)
1. Binary extracts Python to temp dir (via `go-embed-python`)
2. Binary extracts bundled wheels to temp dir
3. `pip install --no-index --find-links <wheels_dir> ansible-core`
4. `ansible-galaxy collection install <ansible.posix tarball>`
5. Playbook runs normally
