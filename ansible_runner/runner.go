package ansible_runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// AnsibleRunner manages Ansible execution within the Go application
type AnsibleRunner struct {
	playbookPath    string
	inventory       string
	privateKeyFile  string
	connection      string
	become          bool
	becomeMethod    string
	verbosity       int
	extraVars       map[string]interface{}
	pythonDir       string
	ansiblePlaybook string
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

// WithExtraVars sets extra variables to pass to the playbook
func WithExtraVars(vars map[string]interface{}) RunnerOption {
	return func(r *AnsibleRunner) { r.extraVars = vars }
}

// WithPythonDir sets the embedded Python directory path
func WithPythonDir(dir string) RunnerOption {
	return func(r *AnsibleRunner) { r.pythonDir = dir }
}

// WithAnsiblePlaybook sets the ansible-playbook binary path
func WithAnsiblePlaybook(path string) RunnerOption {
	return func(r *AnsibleRunner) { r.ansiblePlaybook = path }
}

// PlaybookResult holds the summary of a playbook execution
type PlaybookResult struct {
	Ok            int
	Changed       int
	Failed        int
	Unreachable   int
	Skipped       int
	PlayCount     int
	TaskCount     int
	HostCount     int
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

	// If pythonDir is set but ansiblePlaybook is not, try to find it
	if r.pythonDir != "" && r.ansiblePlaybook == "" {
		r.ansiblePlaybook = findAnsiblePlaybookInPythonDir(r.pythonDir)
	}

	return r, nil
}

// findAnsiblePlaybookInPythonDir searches for ansible-playbook in the embedded Python directory
func findAnsiblePlaybookInPythonDir(pythonDir string) string {
	// Common locations for ansible-playbook in a Python environment
	candidates := []string{
		filepath.Join(pythonDir, "bin", "ansible-playbook"),
		filepath.Join(pythonDir, "Scripts", "ansible-playbook.exe"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Search recursively for ansible-playbook
	var foundPath string
	err := filepath.Walk(pythonDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (info.Name() == "ansible-playbook" || info.Name() == "ansible-playbook.exe") {
			foundPath = path
			return fmt.Errorf("found")
		}
		return nil
	})

	if err != nil && strings.Contains(err.Error(), "found") {
		return foundPath
	}

	return ""
}

// Run executes the Ansible playbook
func (r *AnsibleRunner) Run() (*PlaybookResult, error) {
	output, err := r.RunWithOutput()
	if err != nil {
		return nil, fmt.Errorf("%w\nAnsible output:\n%s", err, output)
	}

	result := parsePlaybookOutput(output)
	return result, nil
}

// RunWithOutput runs the playbook and captures output
func (r *AnsibleRunner) RunWithOutput() (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := r.buildCmd(&stdout, &stderr)

	// Set up environment with embedded Python
	if r.pythonDir != "" {
		r.setupEnv(cmd)
	}

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		return output, fmt.Errorf("playbook execution failed: %w", err)
	}

	return output, nil
}

// buildCmd creates the ansible-playbook command
func (r *AnsibleRunner) buildCmd(stdout, stderr io.Writer) *exec.Cmd {
	ansiblePath := r.ansiblePlaybook
	if ansiblePath == "" {
		ansiblePath = "ansible-playbook"
	}

	var cmd *exec.Cmd

	// Check if it's a Python script (ansible-playbook from site-packages)
	if strings.HasSuffix(ansiblePath, ".py") {
		pythonPath := "python3"
		if r.pythonDir != "" {
			pythonPath = filepath.Join(r.pythonDir, "bin", "python")
			if _, err := os.Stat(pythonPath); err != nil {
				pythonPath = "python3"
			}
		}
		cmd = exec.Command(pythonPath, ansiblePath)
	} else {
		cmd = exec.Command(ansiblePath)
	}

	// Build arguments
	args := []string{r.playbookPath}

	// Connection
	if r.connection != "" && r.connection != "local" {
		args = append(args, "-c", r.connection)
	}

	// Inventory
	if r.inventory != "" {
		args = append(args, "--inventory", r.inventory)
	}

	// Private key
	if r.privateKeyFile != "" {
		args = append(args, "--private-key", r.privateKeyFile)
	}

	// Become
	if r.become {
		args = append(args, "-b")
		if r.becomeMethod != "" && r.becomeMethod != "sudo" {
			args = append(args, "--become-method", r.becomeMethod)
		}
	}

	// Verbosity
	for i := 0; i < r.verbosity; i++ {
		args = append(args, "-v")
	}

	// Extra vars
	for k, v := range r.extraVars {
		args = append(args, "--extra-vars", fmt.Sprintf("%s=%v", k, v))
	}

	cmd.Args = append(cmd.Args, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd
}

// setupAnsibleEnv configures environment variables for the embedded Python/Ansible
func setupAnsibleEnv(cmd *exec.Cmd, pythonDir string) {
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
		if !strings.HasPrefix(env, "PATH=") && !strings.HasPrefix(env, "PYTHONPATH=") {
			newEnv = append(newEnv, env)
		}
	}

	// Add Python bin directory to PATH
	binDir := filepath.Join(pythonDir, "bin")
	if _, err := os.Stat(binDir); err == nil {
		newEnv = append(newEnv, fmt.Sprintf("PATH=%s:%s", binDir, os.Getenv("PATH")))
	} else {
		// Try Scripts dir (Windows)
		binDir = filepath.Join(pythonDir, "Scripts")
		if _, err := os.Stat(binDir); err == nil {
			newEnv = append(newEnv, fmt.Sprintf("PATH=%s;%s", binDir, os.Getenv("PATH")))
		}
	}

	// Set PYTHONPATH to include site-packages
	newEnv = append(newEnv, fmt.Sprintf("PYTHONPATH=%s", pythonDir))

	cmd.Env = newEnv
}

// setupEnv configures environment variables for the embedded Python/Ansible
func (r *AnsibleRunner) setupEnv(cmd *exec.Cmd) {
	setupAnsibleEnv(cmd, r.pythonDir)
}

// parsePlaybookOutput parses ansible-playbook output to extract results
func parsePlaybookOutput(output string) *PlaybookResult {
	result := &PlaybookResult{}

	// Look for the typical ansible summary pattern:
	// PLAY RECAP *********************************************************************
	// localhost                  : ok=10   changed=3    unreachable=0    failed=0    skipped=4    rescued=0    ignored=0

	// Pattern for the summary line per host
	hostPattern := regexp.MustCompile(`(\S+)\s+:\s+ok=(\d+)\s+changed=(\d+)\s+unreachable=(\d+)\s+failed=(\d+)\s+skipped=(\d+)`)

	matches := hostPattern.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) == 6 {
			ok, _ := strconv.Atoi(match[2])
			changed, _ := strconv.Atoi(match[3])
			unreachable, _ := strconv.Atoi(match[4])
			failed, _ := strconv.Atoi(match[5])
			skipped, _ := strconv.Atoi(match[6])

			result.Ok += ok
			result.Changed += changed
			result.Unreachable += unreachable
			result.Failed += failed
			result.Skipped += skipped
		}
	}

	// Try to count plays and tasks from the output
	scanner := bufio.NewScanner(strings.NewReader(output))
	taskCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "TASK [") || strings.Contains(line, "TASK:") {
			taskCount++
		}
	}
	result.TaskCount = taskCount

	// Try JSON output if available (ansible --check with --diff can produce JSON)
	jsonPattern := regexp.MustCompile(`\{[\s\S]*"plays"[\s\S]*\}`)
	if jsonMatch := jsonPattern.FindString(output); jsonMatch != "" {
		var jsonResult struct {
			Plays []struct {
				TaskStats []struct {
					Results []struct {
						Changed bool `json:"changed"`
						Failed  bool `json:"failures"`
					} `json:"hosts"`
				} `json:"task_stats"`
			} `json:"plays"`
		}
		if err := json.Unmarshal([]byte(jsonMatch), &jsonResult); err == nil {
			for _, play := range jsonResult.Plays {
				result.PlayCount++
				for _, task := range play.TaskStats {
					for _, host := range task.Results {
						if host.Changed {
							result.Changed++
						}
						if host.Failed {
							result.Failed++
						} else {
							result.Ok++
						}
					}
				}
			}
		}
	}

	return result
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

// ParsePlaybookOutput parses ansible-playbook output to extract results
// This is exported for testing purposes
func ParsePlaybookOutput(output string) *PlaybookResult {
	return parsePlaybookOutput(output)
}
