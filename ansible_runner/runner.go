package ansible_runner

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apenella/go-ansible/execution"
	"github.com/apenella/go-ansible/playbook"
)

// AnsibleRunner manages Ansible execution within the Go application
type AnsibleRunner struct {
	playbookPath   string
	inventory      string
	privateKeyFile string
	connection     string
	factsDir       string
	varsFiles      []string
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

// WithFactsDir sets the directory for host facts
func WithFactsDir(dir string) RunnerOption {
	return func(r *AnsibleRunner) { r.factsDir = dir }
}

// WithVarsFiles adds variable files to load
func WithVarsFiles(files []string) RunnerOption {
	return func(r *AnsibleRunner) { r.varsFiles = files }
}

// WithVerbosity sets the verbosity level (0-4, -v to -vvvv)
func WithVerbosity(level int) RunnerOption {
	return func(r *AnsibleRunner) { r.verbosity = level }
}

// New creates a new Ansible runner instance
func New(playbookPath string, options ...RunnerOption) (*AnsibleRunner, error) {
	if _, err := os.Stat(playbookPath); err != nil {
		return nil, fmt.Errorf("playbook not found: %w", err)
	}

	r := &AnsibleRunner{
		playbookPath: playbookPath,
		connection:   "local",
	}

	for _, opt := range options {
		opt(r)
	}

	return r, nil
}

// Run executes the Ansible playbook
func (r *AnsibleRunner) Run() (*execution.PlaybookExecutionResult, error) {
	log.Printf("Running Ansible playbook: %s", r.playbookPath)

	// Create playbook instance
	pbOpts := []playbook.Option{
		playbook.WithPlaybook(r.playbookPath),
	}

	if r.inventory != "" {
		pbOpts = append(pbOpts, playbook.WithInventory(r.inventory))
	}
	if r.privateKeyFile != "" {
		pbOpts = append(pbOpts, playbook.WithPrivateKeyFile(r.privateKeyFile))
	}
	if r.factsDir != "" {
		pbOpts = append(pbOpts, playbook.WithFactsCachePath(r.factsDir))
	}

	for _, file := range r.varsFiles {
		pbOpts = append(pbOpts, playbook.WithVarsFiles(file))
	}

	pb, err := playbook.NewPlaybook(pbOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create playbook: %w", err)
	}

	// Set verbosity (1=v, 2=vv, 3=vvv, 4=vvvv=verbose+trace)
	switch r.verbosity {
	case 1:
		pb.SetVerbose(true)
	case 2:
		pb.SetDebug(true)
	case 3:
		pb.SetTrace(true)
	case 4:
		pb.SetVerbose(true)
		pb.SetTrace(true)
	}

	// Create execution context
	execOpts := []execution.Option{
		execution.WithPlaybook(pb),
		execution.WithConnection(r.connection),
	}

	if r.connection == "local" {
		// For local connection, we need to set up the environment properly
		execOpts = append(execOpts, execution.WithRemoteUser("root"))
	}

	exe, err := execution.NewExecution(execOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Run the playbook
	result, err := exe.Run()
	if err != nil {
		return nil, fmt.Errorf("playbook execution failed: %w", err)
	}

	return result, nil
}

// RunWithOutput runs the playbook and captures output
func (r *AnsibleRunner) RunWithOutput() (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("ansible-playbook", "-c", r.connection, r.playbookPath)

	if r.verbosity > 0 {
		vStr := strings.Repeat("v", r.verbosity)
		cmd.Args = append(cmd.Args, "-"+vStr)
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
	)
	if err != nil {
		return err
	}

	result, err := r.Run()
	if err != nil {
		return err
	}

	log.Printf("Playbook completed. Failed: %d, Unreachable: %d, Changed: %d, Ok: %d",
		result.Failed(), result.Unreachable(), result.Changed(), result.Ok())

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
