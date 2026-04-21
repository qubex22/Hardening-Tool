package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

//go:embed authorized_fingerprints.txt
var whitelistFS embed.FS

// authorizedFingerprints is a whitelist of authorized device fingerprint hashes.
// The hash is the raw SHA256 hex string (64 hex chars) as produced by fingerprint.Collect().
// Populated at init() from the embedded authorized_fingerprints.txt file.
// If the file is empty (no fingerprints), all devices are allowed (no license check).
var authorizedFingerprints = map[string]bool{}

func init() {
	data, err := whitelistFS.ReadFile("authorized_fingerprints.txt")
	if err != nil {
		// Fallback: try reading from disk (for development)
		data, err = os.ReadFile("authorized_fingerprints.txt")
		if err != nil {
			// File not found — allow all devices
			return
		}
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip sha256: prefix if present
		hash := strings.TrimPrefix(line, "sha256:")
		authorizedFingerprints[hash] = true
	}
}

// EncryptedAssets holds information about encrypted playbook data
type EncryptedAssets struct {
	PlaybookData  []byte
	KeyDerivation string // Method used: "hmac-sha256"
}

// IsAuthorized checks if the device fingerprint is on the whitelist
func IsAuthorized(fingerprintHash string) bool {
	// If no fingerprints are configured, allow all (no license check)
	if len(authorizedFingerprints) == 0 {
		return true
	}
	_, authorized := authorizedFingerprints[fingerprintHash]
	return authorized
}

// VerifyFingerprint validates a fingerprint against the whitelist and returns appropriate error info
func VerifyFingerprint(fingerprintHash string) (bool, string) {
	// Normalize: strip "sha256:" prefix if present
	hash := strings.TrimPrefix(fingerprintHash, "sha256:")
	if !IsAuthorized(hash) {
		return false, fmt.Sprintf(
			"Error: Unauthorized device.\n"+
				"Fingerprint hash: %s\n"+
				"Please contact support@yourorg.com to request authorization.",
			hash,
		)
	}
	return true, ""
}

// DeriveKey derives a device-specific encryption key from the fingerprint
func DeriveKey(fingerprint string, masterSecret []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, masterSecret)
	mac.Write([]byte(fingerprint))
	return mac.Sum(nil), nil
}

// DecryptAESGCM decrypts data using AES-256-GCM with a derived key
func DecryptAESGCM(encryptedData string, fingerprint string, masterSecret []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	key, err := DeriveKey(fingerprint, masterSecret)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize+12 {
		return nil, fmt.Errorf("encrypted data too short")
	}
	nonce := data[:12]
	ciphertext := data[12:]

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptAESGCM encrypts data using AES-256-GCM with a derived key
func EncryptAESGCM(plaintext []byte, fingerprint string, masterSecret []byte) (string, error) {
	key, err := DeriveKey(fingerprint, masterSecret)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, len(nonce)+len(ciphertext))
	copy(result[:12], nonce)
	copy(result[12:], ciphertext)

	return base64.StdEncoding.EncodeToString(result), nil
}

// GetWhitelist returns the current whitelist (for diagnostics)
func GetWhitelist() []string {
	var hashes []string
	for h := range authorizedFingerprints {
		hashes = append(hashes, h)
	}
	return hashes
}

// AddAuthorizedHash adds a hash to the whitelist at runtime
// Note: This is for testing/diagnostics only. In production, use static whitelisting.
func AddAuthorizedHash(hash string) {
	hash = strings.TrimPrefix(hash, "sha256:")
	authorizedFingerprints[hash] = true
}
