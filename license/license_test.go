package license

import (
	"testing"
)

func TestIsAuthorizedEmptyWhitelist(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear the map to simulate empty whitelist
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}

	// With empty whitelist, all devices should be allowed
	authorized := IsAuthorized("0000000000000000000000000000000000000000000000000000000000000000")
	if !authorized {
		t.Error("Empty whitelist should allow all devices")
	}
}

func TestIsAuthorizedWithHash(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear and add a test hash
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}
	testHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	authorizedFingerprints[testHash] = true

	// Test authorized hash
	if !IsAuthorized(testHash) {
		t.Error("Hash in whitelist should be authorized")
	}

	// Test unauthorized hash
	if IsAuthorized("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
		t.Error("Hash not in whitelist should be unauthorized")
	}
}

func TestVerifyFingerprintEmptyWhitelist(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear the map to simulate empty whitelist
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}

	// With empty whitelist, any fingerprint should be valid
	hash := "0000000000000000000000000000000000000000000000000000000000000000"
	valid, _ := VerifyFingerprint(hash)
	if !valid {
		t.Error("Empty whitelist should allow any fingerprint")
	}
}

func TestVerifyFingerprintUnauthorized(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear and add a hash
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}
	authorizedFingerprints["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"] = true

	hash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	valid, msg := VerifyFingerprint(hash)

	if valid {
		t.Error("Should return invalid for unauthorized fingerprint")
	}

	if msg == "" {
		t.Error("Error message should not be empty")
	}
}

func TestVerifyFingerprintWithPrefix(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear and add a hash
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}
	testHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	authorizedFingerprints[testHash] = true

	// Test with sha256: prefix — should still match
	valid, _ := VerifyFingerprint("sha256:" + testHash)
	if !valid {
		t.Error("sha256: prefix should be stripped and still match")
	}
}

func TestAddAuthorizedHash(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear the map
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}

	testHash := "testhash123abcdef"
	AddAuthorizedHash(testHash)

	if !IsAuthorized(testHash) {
		t.Error("Hash should have been added to whitelist")
	}
}

func TestGetWhitelist(t *testing.T) {
	// Save original state and restore after test
	original := make(map[string]bool)
	for k, v := range authorizedFingerprints {
		original[k] = v
	}
	defer func() {
		for k := range authorizedFingerprints {
			delete(authorizedFingerprints, k)
		}
		for k, v := range original {
			authorizedFingerprints[k] = v
		}
	}()

	// Clear and add test hashes
	for k := range authorizedFingerprints {
		delete(authorizedFingerprints, k)
	}
	authorizedFingerprints["aaaa"] = true
	authorizedFingerprints["bbbb"] = true

	whitelist := GetWhitelist()
	if len(whitelist) != 2 {
		t.Errorf("Expected 2 hashes in whitelist, got %d", len(whitelist))
	}
}
