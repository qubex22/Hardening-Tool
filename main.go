package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"harden_sles15/ansible_runner"
	"harden_sles15/fingerprint"
	"harden_sles15/license"
	"harden_sles15/python"
)

//go:embed ansible/playbook.yml ansible/roles
var playbookFS embed.FS

// extractEmbeddedDir extracts a directory from the embedded filesystem to disk
func extractEmbeddedDir(f embed.FS, src, dst string) error {
	src = filepath.ToSlash(src)
	return fs.WalkDir(f, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, filepath.ToSlash(path))
		if err != nil {
			return err
		}
		outPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}
		content, err := f.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(outPath, content, 0644)
	})
}

func main() {
	var level int
	flag.IntVar(&level, "level", 1, "Hardening level (0=disabled, 1=basic, 2=medium, 3=maximum)")
	flag.Parse()

	if level < 0 || level > 3 {
		log.Fatalf("Invalid hardening level: %d (must be 0-3)", level)
	}

	log.Printf("Hardening level: %d", level)

	log.Println("========================================")
	log.Println("  SLES 15 Hardening Tool v1.0.0")
	log.Println("========================================")

	// Initialize embedded Python runtime
	log.Println("\n[Initializing embedded Python runtime...]")
	pyRuntime, err := python.New()
	if err != nil {
		log.Fatalf("Failed to initialize embedded Python: %v", err)
	}
	defer pyRuntime.Cleanup()

	if err := pyRuntime.Verify(); err != nil {
		log.Fatalf("Python verification failed: %v", err)
	}
	log.Println("Embedded Python runtime ready.")

	pythonDir := pyRuntime.GetPythonDir()
	ansiblePlaybookPath := pyRuntime.GetAnsiblePlaybookPath()

	if ansiblePlaybookPath == "" {
		log.Fatalf("ansible-playbook not found in embedded Python assets. " +
			"Please ensure ansible is installed in the build assets.")
	}
	log.Printf("ansible-playbook path: %s", ansiblePlaybookPath)

	// Step 1: Collect device fingerprint
	log.Println("\n[Step 1/4] Collecting device fingerprint...")
	fp, err := fingerprint.Collect()
	if err != nil {
		log.Fatalf("Failed to collect fingerprint: %v", err)
	}
	log.Printf("Device fingerprint collected.")
	log.Printf("Hash: %s\n", fp.GetHash())

	// Step 2: Validate license/authorization
	log.Println("\n[Step 2/4] Validating license...")
	authorized, msg := license.VerifyFingerprint(fp.Hash)
	if !authorized {
		log.Fatalf("%s\n\nThis device is not authorized for hardening.\n"+
			"To request authorization, submit this fingerprint to support:\n%s",
			msg, fp.GetHash())
	}
	log.Println("License validation successful.")

	// Step 3: Extract embedded playbook and roles
	log.Println("\n[Step 3/4] Preparing embedded assets...")
	playbookContent, err := playbookFS.ReadFile("ansible/playbook.yml")
	if err != nil {
		log.Fatalf("Failed to read embedded playbook: %v", err)
	}

	// Write playbook to temporary location for execution
	tempDir, err := os.MkdirTemp("", "harden_*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	playbookPath := filepath.Join(tempDir, "playbook.yml")
	if err := os.WriteFile(playbookPath, playbookContent, 0644); err != nil {
		log.Fatalf("Failed to write playbook: %v", err)
	}

	// Extract roles directory
	rolesDir := filepath.Join(tempDir, "roles")
	if err := extractEmbeddedDir(playbookFS, "ansible/roles", rolesDir); err != nil {
		log.Fatalf("Failed to extract roles: %v", err)
	}
	log.Printf("Roles extracted to: %s", rolesDir)

	// Step 4: Run Ansible playbook
	log.Println("\n[Step 4/4] Running hardening playbook...")
	if err := runHardening(playbookPath, level, pythonDir, ansiblePlaybookPath); err != nil {
		log.Fatalf("Playbook execution failed: %v", err)
	}

	log.Println("\n========================================")
	log.Println("  Hardening completed successfully!")
	log.Println("========================================")

	// Final fingerprint verification
	finalFp, err := fingerprint.Collect()
	if err != nil {
		log.Printf("Warning: Could not verify final fingerprint: %v", err)
	} else if finalFp.Hash == fp.Hash {
		log.Printf("\nSystem identity unchanged (expected).")
	}

	fmt.Printf("\nFinal fingerprint for compliance report:\n%s\n", fp.GetHash())
}

func runHardening(playbookPath string, level int, pythonDir string, ansiblePlaybookPath string) error {
	// Validate playbook
	if err := ansible_runner.ValidatePlaybook(playbookPath); err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	// Run the embedded playbook with local connection
	r, err := ansible_runner.New(playbookPath,
		ansible_runner.WithConnection("local"),
		ansible_runner.WithVerbosity(2), // -vv
		ansible_runner.WithExtraVars(map[string]interface{}{
			"hardening_level": strconv.Itoa(level),
		}),
		ansible_runner.WithPythonDir(pythonDir),
		ansible_runner.WithAnsiblePlaybook(ansiblePlaybookPath),
	)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}
	result, err := r.Run()

	if err != nil {
		return fmt.Errorf("playbook execution failed: %w", err)
	}

	// Log results
	log.Printf("Playbook Results:")
	log.Printf("  Failed:      %d", result.Failed)
	log.Printf("  Unreachable: %d", result.Unreachable)
	log.Printf("  Changed:     %d", result.Changed)
	log.Printf("  Ok:          %d", result.Ok)

	// Check for failures
	if result.Failed > 0 || result.Unreachable > 0 {
		return fmt.Errorf("playbook had %d failed tasks and %d unreachable hosts",
			result.Failed, result.Unreachable)
	}

	return nil
}
