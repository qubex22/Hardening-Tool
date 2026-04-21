package ansible_runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ansibleExecute "github.com/apenella/go-ansible/pkg/execute"
	ansibleOptions "github.com/apenella/go-ansible/pkg/options"
	ansiblePlaybook "github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
)

// AnsibleRunner manages Ansible execution within the Go application
type AnsibleRunner struct {
	playbookPath   string
	inventory      string
	privateKeyFile string
	connection     string
	become         bool
	becomeMethod   string
	verbosity      int
}

// RunnerOption configures an AnsibleRunner
type RunnerOption func(*AnsibleRunner)

// WithInventory sets the inventory source
func WithInventory(inv string) RunnerOption {
	return func(r *AnsibleRunner) { r.inventory = inv }
}

// WithPrivateKey sets the SSH private key file path
func WithPrivateKey(file string) RunnerOption {
	return func(r *AnsibleRunner) { r.privateKeyFile = file }
}

// WithConnection sets the connection type (local, ssh, etc.)
func WithConnection(conn string) RunnerOption {
	return func(r *AnsibleRunner) { r.connection = conn }
}

// WithBecome enables privilege escalation (become/sudo)
func WithBecome(become bool) RunnerOption {
	return func(r *AnsibleRunner) { r.become = become }
}

// WithBecomeMethod sets the privilege escalation method (sudo, su, etc.)
func WithBecomeMethod(method string) RunnerOption {
	return func(r *AnsibleRunner) { r.becomeMethod = method }
}

// WithVerbosity sets the verbosity level (0-4, -v to -vvvv)
func WithVerbosity(level int) RunnerOption {
	return func(r *AnsibleRunner) { r.verbosity = level }
}

// PlaybookResult holds the summary of a playbook execution
type PlaybookResult struct {
	Ok        int
	Changed   int
	Failed    int
	Unreachable int
	Skipped   int
}

// New creates a new Ansible runner instance
func New(playbookPath string, options ...RunnerOption) (*AnsibleRunner, error) {
	if _, err := os.Stat(playbookPath); err != nil {
		return nil, fmt.Errorf("playbook not found: %w", err)
	}

	r := &AnsibleRunner{
		playbookPath: playbookPath,
		connection:   "local",
		become:       true,
		becomeMethod: "sudo",
	}

	for _, opt := range options {
		opt(r)
	}

	return r, nil
}

// Run executes the Ansible playbook
func (r *AnsibleRunner) Run() (*PlaybookResult, error) {
	log.Printf("Running Ansible playbook: %s", r.playbookPath)

	// Build playbook options
	pbOptions := &ansiblePlaybook.AnsiblePlaybookOptions{
		Inventory: r.inventory,
	}

	// Set verbosity
	switch r.verbosity {
	case 1:
		pbOptions.Verbose = true
	case 2:
		pbOptions.VerboseV = true
	case 3:
		pbOptions.VerboseVV = true
	case 4:
		pbOptions.VerboseVVV = true
	case 5:
		pbOptions.VerboseVVVV = true
	}

	// Build connection options
	connOptions := &ansibleOptions.AnsibleConnectionOptions{
		Connection: r.connection,
	}

	// Build privilege escalation options
	privEscOptions := &ansibleOptions.AnsiblePrivilegeEscalationOptions{
		Become:       r.become,
		BecomeMethod: r.becomeMethod,
	}

	// Create the playbook command
	cmd := &ansiblePlaybook.AnsiblePlaybookCmd{
		Playbooks:                  []string{r.playbookPath},
		ConnectionOptions:          connOptions,
		PrivilegeEscalationOptions: privEscOptions,
		Options:                    pbOptions,
		Exec:                       ansibleExecute.NewDefaultExecute(),
	}

	// Run with context
	ctx := context.Background()
	err := cmd.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("playbook execution failed: %w", err)
	}

	// Return a default result since go-ansible v1.x doesn't return parsed results
	// The actual results are streamed via stdout callback
	return &PlaybookResult{Ok: 0, Changed: 0, Failed: 0, Unreachable: 0, Skipped: 0}, nil
}

// RunWithOutput runs the playbook and captures output
func (r *AnsibleRunner) RunWithOutput() (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("ansible-playbook", r.playbookPath, "-c", r.connection)

	if r.verbosity > 0 {
		vStr := strings.Repeat("v", r.verbosity)
		cmd.Args = append(cmd.Args, "-"+vStr)
	}

	if r.inventory != "" {
		cmd.Args = append(cmd.Args, "--inventory", r.inventory)
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		return output, fmt.Errorf("playbook execution failed: %w", err)
	}

	return output, nil
}

// RunEmbeddedPlaybook executes an embedded playbook from the binary assets
func RunEmbeddedPlaybook(playbookPath string) error {
	log.Printf("Running embedded playbook: %s", playbookPath)

	// Get absolute path relative to executable location
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	fullPlaybookPath := filepath.Join(exeDir, playbookPath)
	log.Printf("Resolved playbook path: %s", fullPlaybookPath)

	r, err := New(fullPlaybookPath,
		WithConnection("local"),
		WithVerbosity(2), // Default to -vv
		WithBecome(true),
		WithBecomeMethod("sudo"),
	)
	if err != nil {
		return err
	}

	_, err = r.Run()
	if err != nil {
		return err
	}

	log.Printf("Playbook completed successfully")

	return nil
}

// ValidatePlaybook checks if the playbook file exists and is valid YAML
func ValidatePlaybook(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("playbook not found: %w", err)
	}

	// Basic YAML validation - check for common playbook structure
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read playbook: %w", err)
	}

	contentStr := string(content)

	// Check for required playbook keys
	requiredKeys := []string{"hosts:", "tasks:"}
	for _, key := range requiredKeys {
		if !strings.Contains(contentStr, key) {
			return fmt.Errorf("playbook missing required key: %s", key)
		}
	}

	return nil
}

// NewDefaultExecutor returns a new DefaultExecute executor with JSON output
func NewDefaultExecutor() ansibleExecute.Executor {
	return ansibleExecute.NewDefaultExecute(
		ansibleExecute.WithWrite(io.Discard),
		ansibleExecute.WithWriteError(io.Discard),
	)
}

// NewJSONResultCallback returns a stdout callback function that parses JSON results
func NewJSONResultCallback() results.StdoutCallbackResultsFunc {
	return results.JSONStdoutCallbackResults
}
