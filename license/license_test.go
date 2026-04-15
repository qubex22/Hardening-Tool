package license

import (
	"testing"
)

func TestIsAuthorized(t *testing.T) {
	// Test with empty whitelist (should all be unauthorized)
	hash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	authorized := IsAuthorized(hash)
	if authorized {
		t.Error("Empty whitelist should not authorize any hash")
	}
}

func TestVerifyFingerprintUnauthorized(t *testing.T) {
	hash := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	valid, msg := VerifyFingerprint(hash)

	if valid {
		t.Error("Should return invalid for unauthorized fingerprint")
	}

	if msg == "" {
		t.Error("Error message should not be empty")
	}
}

func TestAddAuthorizedHash(t *testing.T) {
	testHash := "test-hash-123"
	AddAuthorizedHash(testHash)

	if !IsAuthorized(testHash) && !IsAuthorized("sha256:"+testHash) {
		t.Error("Hash should have been added to whitelist")
	}
}

func TestDeriveKey(t *testing.T) {
	fingerprint := "test-fingerprint"
	masterSecret := []byte("super-secret-master-key")

	key1, err := DeriveKey(fingerprint, masterSecret)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	if len(key1) == 0 {
		t.Error("Derived key should not be empty")
	}

	// Same inputs should produce same output (deterministic)
	key2, err := DeriveKey(fingerprint, masterSecret)
	if err != nil {
		t.Fatalf("Failed to derive key second time: %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("Key derivation should be deterministic")
	}
}

func TestDeriveKeyDifferentInputs(t *testing.T) {
	fingerprint := "test-fingerprint"
	masterSecret := []byte("super-secret-master-key")

	key1, _ := DeriveKey(fingerprint, masterSecret)

	// Different fingerprint should produce different key
	key2, _ := DeriveKey("different-fingerprint", masterSecret)

	if string(key1) == string(key2) {
		t.Error("Different fingerprints should produce different keys")
	}
}
