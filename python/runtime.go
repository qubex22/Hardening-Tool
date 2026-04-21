package python

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kluctl/go-embed-python/python"
)

// PythonRuntime manages the embedded Python environment
type PythonRuntime struct {
	ep *python.EmbeddedPython
}

// New initializes the embedded Python runtime and installs ansible if needed
func New() (*PythonRuntime, error) {
	ep, err := python.NewEmbeddedPython("harden_sles15")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedded Python: %w", err)
	}

	r := &PythonRuntime{ep: ep}

	// Install ansible-core into the embedded Python if not already present
	if err := r.ensureAnsible(); err != nil {
		return nil, fmt.Errorf("failed to install ansible: %w", err)
	}

	return r, nil
}

// ensureAnsible installs ansible-core into the embedded Python runtime
func (r *PythonRuntime) ensureAnsible() error {
	pythonExe := r.ep.GetExePath()
	if _, err := os.Stat(pythonExe); err != nil {
		return fmt.Errorf("embedded Python not found at %s", pythonExe)
	}

	// Check if ansible is already installed
	checkCmd := exec.Command(pythonExe, "-c", "import ansible; print(ansible.__version__)")
	if output, err := checkCmd.CombinedOutput(); err == nil {
		fmt.Printf("Ansible %s already installed.\n", strings.TrimSpace(string(output)))
		return nil
	}

	// Check if pip exists
	pipCheck := exec.Command(pythonExe, "-m", "pip", "--version")
	if pipCheck.Run() == nil {
		fmt.Println("pip already available.")
	} else {
		// Download get-pip.py and bootstrap pip
		fmt.Println("Bootstrapping pip via get-pip.py...")
		getPipUrl := "https://bootstrap.pypa.io/get-pip.py"
		getPipCmd := exec.Command(pythonExe, "-c", fmt.Sprintf(
			"import urllib.request; urllib.request.urlretrieve(%q, '/tmp/get-pip.py')", getPipUrl))
		if _, err := getPipCmd.CombinedOutput(); err != nil {
			// Try http fallback
			getPipUrl = "http://bootstrap.pypa.io/get-pip.py"
			getPipCmd = exec.Command(pythonExe, "-c", fmt.Sprintf(
				"import urllib.request; urllib.request.urlretrieve(%q, '/tmp/get-pip.py')", getPipUrl))
			if _, err := getPipCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to download get-pip.py: %w", err)
			}
		}
		installPipCmd := exec.Command(pythonExe, "/tmp/get-pip.py", "--quiet")
		if output, err := installPipCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install pip: %w: %s", err, string(output))
		}
		fmt.Println("pip bootstrapped successfully.")
	}

	// Install ansible-core
	fmt.Println("Installing ansible-core...")
	installCmd := exec.Command(pythonExe, "-m", "pip", "install", "--quiet", "ansible-core")
	if output, err := installCmd.CombinedOutput(); err != nil {
		fmt.Printf("pip install output: %s\n", string(output))
		return fmt.Errorf("pip install ansible-core failed: %w", err)
	}

	fmt.Println("Ansible installed successfully.")
	return nil
}

// RunScript executes a Python script using the embedded runtime
func (r *PythonRuntime) RunScript(script string, args []string) error {
	// Create a temporary script file
	tmpFile, err := os.CreateTemp("", "harden_*.py")
	if err != nil {
		return fmt.Errorf("failed to create temp script: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(script)); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	// Build Python command with embedded path
	cmdArgs := append([]string{tmpFile.Name()}, args...)

	cmd := r.ep.PythonCmd(cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}

	return nil
}

// GetPythonDir returns the path to the embedded Python installation
func (r *PythonRuntime) GetPythonDir() string {
	return r.ep.GetExtractedPath()
}

// GetPythonExe returns the path to the embedded Python executable
func (r *PythonRuntime) GetPythonExe() string {
	return r.ep.GetExePath()
}

// Verify checks if Python is properly initialized
func (r *PythonRuntime) Verify() error {
	pythonBin := r.ep.GetExePath()
	if _, err := os.Stat(pythonBin); err != nil {
		return fmt.Errorf("embedded Python binary not found at %s: %w", pythonBin, err)
	}
	return nil
}

// GetAnsiblePlaybookPath returns the path to ansible-playbook in the embedded Python environment
func (r *PythonRuntime) GetAnsiblePlaybookPath() string {
	extractedPath := r.ep.GetExtractedPath()

	// Try common locations
	candidates := []string{
		filepath.Join(extractedPath, "bin", "ansible-playbook"),
		filepath.Join(extractedPath, "Scripts", "ansible-playbook.exe"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Search for ansible-playbook.py in site-packages
	var sitePackagesPattern string
	if runtime.GOOS == "windows" {
		sitePackagesPattern = filepath.Join(extractedPath, "lib", "site-packages")
	} else {
		sitePackagesPattern = filepath.Join(extractedPath, "lib", "python*", "site-packages")
	}

	// Check if the pattern directory exists (glob may not work with os.Stat)
	if _, err := os.Stat(sitePackagesPattern); err == nil {
		var foundPath string
		err := filepath.Walk(filepath.Dir(sitePackagesPattern), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && info.Name() == "site-packages" {
				ansiblePath := filepath.Join(path, "ansible", "cli", "ansible_playbook.py")
				if _, err := os.Stat(ansiblePath); err == nil {
					foundPath = ansiblePath
					return fmt.Errorf("found")
				}
			}
			return nil
		})
		if err == nil || foundPath != "" {
			if foundPath != "" {
				return foundPath
			}
		}
	}

	// Also try walking the entire extracted path as fallback
	var finalPath string
	filepath.Walk(extractedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "ansible_playbook.py" {
			finalPath = path
			return fmt.Errorf("found")
		}
		return nil
	})
	if finalPath != "" {
		return finalPath
	}

	return ""
}

// SetupAnsibleEnv configures environment variables for running ansible with embedded Python
func SetupAnsibleEnv(cmd *exec.Cmd, pythonDir string) {
	if pythonDir == "" {
		return
	}

	// Get existing env or use current process env
	currentEnv := cmd.Env
	if len(currentEnv) == 0 {
		currentEnv = os.Environ()
	}

	// Build new env with Python paths
	var newEnv []string
	for _, env := range currentEnv {
		// Remove any existing PATH/PYTHONPATH that might conflict
		if !startsWith(env, "PATH=") && !startsWith(env, "PYTHONPATH=") {
			newEnv = append(newEnv, env)
		}
	}

	// Add Python bin directory to PATH
	var binDir string
	if runtime.GOOS == "windows" {
		binDir = filepath.Join(pythonDir, "Scripts")
	} else {
		binDir = filepath.Join(pythonDir, "bin")
	}

	if _, err := os.Stat(binDir); err == nil {
		pathEnv := os.Getenv("PATH")
		if runtime.GOOS == "windows" {
			newEnv = append(newEnv, fmt.Sprintf("PATH=%s;%s", binDir, pathEnv))
		} else {
			newEnv = append(newEnv, fmt.Sprintf("PATH=%s:%s", binDir, pathEnv))
		}
	}

	// Set PYTHONPATH to include the extracted Python directory
	newEnv = append(newEnv, fmt.Sprintf("PYTHONPATH=%s", pythonDir))

	// Set LD_LIBRARY_PATH for Linux so embedded Python can find its libraries
	if runtime.GOOS == "linux" {
		ldPath := os.Getenv("LD_LIBRARY_PATH")
		libDir := filepath.Join(pythonDir, "lib")
		if _, err := os.Stat(libDir); err == nil {
			if ldPath != "" {
				newEnv = append(newEnv, fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", libDir, ldPath))
			} else {
				newEnv = append(newEnv, fmt.Sprintf("LD_LIBRARY_PATH=%s", libDir))
			}
		}
	}

	cmd.Env = newEnv
}

// RunAnsiblePlaybook runs an Ansible playbook using the embedded Python/Ansible
func (r *PythonRuntime) RunAnsiblePlaybook(playbookPath string, connection string) error {
	script := fmt.Sprintf(`
import sys
import os

# Ensure we can find ansible
python_path = %q
sys.path.insert(0, python_path)

try:
    import ansible
    print("Ansible version: " + ansible.__version__)
except ImportError as e:
    print("Failed to import Ansible: " + str(e), file=sys.stderr)
    sys.exit(1)

# Run the playbook
import subprocess
result = subprocess.run(
    ["ansible-playbook", "-c", %q, %q],
    stdout=sys.stdout,
    stderr=sys.stderr
)
sys.exit(result.returncode)
`, r.GetPythonDir(), connection, playbookPath)

	return r.RunScript(script, []string{})
}

// Cleanup releases embedded Python resources
func (r *PythonRuntime) Cleanup() {
	if r.ep != nil {
		r.ep.Cleanup()
	}
}

// startsWith checks if a string starts with a given prefix (handles "KEY=value" format)
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
