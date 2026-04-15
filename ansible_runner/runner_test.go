package ansible_runner

import (
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
