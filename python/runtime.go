//go:embed assets/*
package python

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kluctl/go-embed-python/v2/embed"
)

// PythonRuntime manages the embedded Python environment
type PythonRuntime struct {
	py        *embed.EmbeddedPython
	pythonDir string
}

// New initializes the embedded Python runtime
func New() (*PythonRuntime, error) {
	e, err := embed.New(embed.WithPythonVersion("3.11"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedded Python: %w", err)
	}

	return &PythonRuntime{
		py:        e,
		pythonDir: e.PythonDir(),
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
	pythonPath := filepath.Join(r.pythonDir, "bin", "python3")
	cmdArgs := append([]string{tmpFile.Name()}, args...)

	// Add the script directory to PYTHONPATH
	if err := r.py.RunCommand(cmdArgs...); err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}

	return nil
}

// GetPythonDir returns the path to the embedded Python installation
func (r *PythonRuntime) GetPythonDir() string {
	return r.pythonDir
}

// Verify checks if Python is properly initialized
func (r *PythonRuntime) Verify() error {
	pythonBin := filepath.Join(r.pythonDir, "bin", "python3")
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
    print(f"Ansible version: {ansible.__version__}")
except ImportError as e:
    print(f"Failed to import Ansible: {{e}}", file=sys.stderr)
    sys.exit(1)

# Run the playbook
from ansible.runner import Runner
runner = Runner(playbook='{{playbookPath}}', connection='{{connection}}')
result = runner.run()
if result == 0:
    print("Playbook completed successfully")
else:
    print(f"Playbook failed with exit code {{result}}")
    sys.exit(1)
`, r.pythonDir, playbookPath, connection)

	return r.RunScript(script, []string{})
}
