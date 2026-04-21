package ansible_runner

import (
	"os"
	"testing"
)

func TestValidatePlaybook(t *testing.T) {
	// Create a temporary playbook file for testing
	playbookContent := `---
- name: Test playbook
  hosts: localhost
  gather_facts: false
  tasks:
    - debug:
        msg: "Hello from test playbook"
`

	tmpFile, err := os.CreateTemp("", "test_playbook_*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(playbookContent)); err != nil {
		t.Fatalf("Failed to write playbook: %v", err)
	}

	// Test validation
	err = ValidatePlaybook(tmpFile.Name())
	if err != nil {
		t.Errorf("Valid playbook failed validation: %v", err)
	}
}

func TestValidatePlaybookMissingKeys(t *testing.T) {
	invalidPlaybook := `---
# Missing hosts and tasks - invalid playbook
some_key:
  nested_key: value
`

	tmpFile, err := os.CreateTemp("", "invalid_playbook_*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(invalidPlaybook)); err != nil {
		t.Fatalf("Failed to write playbook: %v", err)
	}

	err = ValidatePlaybook(tmpFile.Name())
	if err == nil {
		t.Error("Expected validation error for invalid playbook")
	}
}

func TestValidatePlaybookNotFound(t *testing.T) {
	err := ValidatePlaybook("/nonexistent/playbook.yml")
	if err == nil {
		t.Error("Expected error for nonexistent playbook")
	}
}

func TestParsePlaybookOutput(t *testing.T) {
	testOutput := `
PLAY [Apply SLES 15 Security Hardening] *****************************************

TASK [Gathering Facts] **********************************************************
ok: [localhost]

TASK [Verify system is SLES 15] *************************************************
changed: [localhost]

TASK [sles15_cis : Apply basic hardening] ***************************************
changed: [localhost]

PLAY RECAP *********************************************************************
localhost                  : ok=3    changed=2    unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
`

	result := ParsePlaybookOutput(testOutput)

	if result.Ok != 3 {
		t.Errorf("Expected 3 ok, got %d", result.Ok)
	}
	if result.Changed != 2 {
		t.Errorf("Expected 2 changed, got %d", result.Changed)
	}
	if result.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", result.Failed)
	}
	if result.Unreachable != 0 {
		t.Errorf("Expected 0 unreachable, got %d", result.Unreachable)
	}
	if result.Skipped != 1 {
		t.Errorf("Expected 1 skipped, got %d", result.Skipped)
	}
}

func TestParsePlaybookOutputWithFailure(t *testing.T) {
	testOutput := `
PLAY RECAP *********************************************************************
localhost                  : ok=2    changed=1    unreachable=1    failed=1    skipped=0    rescued=0    ignored=0
`

	result := ParsePlaybookOutput(testOutput)

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", result.Failed)
	}
	if result.Unreachable != 1 {
		t.Errorf("Expected 1 unreachable, got %d", result.Unreachable)
	}
}

func TestParsePlaybookOutputEmpty(t *testing.T) {
	result := ParsePlaybookOutput("")

	if result.Ok != 0 || result.Changed != 0 || result.Failed != 0 {
		t.Errorf("Expected all zeros for empty output, got: %+v", result)
	}
}

func TestParsePlaybookOutputMultipleHosts(t *testing.T) {
	testOutput := `
PLAY RECAP *********************************************************************
node1.example.com          : ok=5    changed=2    unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
node2.example.com          : ok=5    changed=2    unreachable=0    failed=0    skipped=1    rescued=0    ignored=0
`

	result := ParsePlaybookOutput(testOutput)

	if result.Ok != 10 {
		t.Errorf("Expected 10 ok (5+5), got %d", result.Ok)
	}
	if result.Changed != 4 {
		t.Errorf("Expected 4 changed (2+2), got %d", result.Changed)
	}
}
