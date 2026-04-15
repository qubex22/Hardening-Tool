package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yourorg/harden-sles15/ansible_runner"
	"github.com/yourorg/harden-sles15/fingerprint"
	"github.com/yourorg/harden-sles15/license"
)

//go:embed ansible/playbook.yml
var playbookFS embed.FS

func main() {
	log.Println("========================================")
	log.Println("  SLES 15 Hardening Tool v1.0.0")
	log.Println("========================================")

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

	// Step 3: Extract embedded playbook
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
	log.Printf("Playbook extracted to: %s", playbookPath)

	// Step 4: Run Ansible playbook
	log.Println("\n[Step 4/4] Running hardening playbook...")
	if err := runHardening(playbookPath); err != nil {
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

func runHardening(playbookPath string) error {
	// Validate playbook
	if err := ansible_runner.ValidatePlaybook(playbookPath); err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	// Run the embedded playbook with local connection
	result, err := ansible_runner.New(playbookPath,
		ansible_runner.WithConnection("local"),
		ansible_runner.WithVerbosity(2), // -vv
	).Run()

	if err != nil {
		return fmt.Errorf("playbook execution failed: %w", err)
	}

	// Log results
	log.Printf("Playbook Results:")
	log.Printf("  Failed:    %d", result.Failed())
	log.Printf("  Unreachable: %d", result.Unreachable())
	log.Printf("  Changed:   %d", result.Changed())
	log.Printf("  Ok:        %d", result.Ok())

	// Check for failures
	if result.Failed() > 0 || result.Unreachable() > 0 {
		return fmt.Errorf("playbook had %d failed tasks and %d unreachable hosts",
			result.Failed(), result.Unreachable())
	}

	return nil
}
