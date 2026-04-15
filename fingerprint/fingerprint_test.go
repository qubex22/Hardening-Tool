package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCollectAndHash(t *testing.T) {
	hash, err := CollectAndHash()
	if err != nil {
		t.Fatalf("Failed to collect fingerprint: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify it's a valid hex SHA256 (64 characters)
	if len(hash) != 64 {
		t.Errorf("Expected 64 char hash, got %d", len(hash))
	}
}

func TestNormalizedOutput(t *testing.T) {
	fp := &Fingerprint{
		MachineID:     "test-machine-id",
		ProductUUID:   "test-uuid",
		ProductSerial: "",
	}

	fp.buildNormalized()

	if !strings.Contains(fp.Normalized, "machine-id:test-machine-id") {
		t.Errorf("Expected machine-id in normalized output")
	}
	if !strings.Contains(fp.Normalized, "product-uuid:test-uuid") {
		t.Errorf("Expected product-uuid in normalized output")
	}
}

func TestDeterministicHash(t *testing.T) {
	// Two calls should produce the same hash for the same input
	hash1 := sha256.Sum256([]byte("test-input"))
	hash2 := sha256.Sum256([]byte("test-input"))

	if hex.EncodeToString(hash1[:]) != hex.EncodeToString(hash2[:]) {
		t.Error("Hash should be deterministic")
	}
}

func TestFileReadTrim(t *testing.T) {
	// Test with empty file path (should return empty string, not error)
	result := readFileTrim("/nonexistent/path")
	if result != "" {
		t.Errorf("Expected empty string for nonexistent file, got: %s", result)
	}
}

func TestDMIRead(t *testing.T) {
	// This will fail on non-Linux systems or without DMI
	result := readDMIFile("product_uuid")
	// Just verify it doesn't panic and returns a string
	if len(result) > 0 {
		t.Logf("DMI product_uuid found: %s", result)
	}
}
