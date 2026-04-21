//go:embed assets/*
package python

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kluctl/go-embed-python/python"
)

// PythonRuntime manages the embedded Python environment
type PythonRuntime struct {
	ep *python.EmbeddedPython
}

// New initializes the embedded Python runtime
func New() (*PythonRuntime, error) {
	ep, err := python.NewEmbeddedPython("harden-sles15")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedded Python: %w", err)
	}

	return &PythonRuntime{
		ep: ep,
	}, nil
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

// Verify checks if Python is properly initialized
func (r *PythonRuntime) Verify() error {
	pythonBin := r.ep.GetExePath()
	if _, err := os.Stat(pythonBin); err != nil {
		return fmt.Errorf("embedded Python binary not found at %s: %w", pythonBin, err)
	}
	return nil
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
