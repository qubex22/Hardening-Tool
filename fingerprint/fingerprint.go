package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Fingerprint represents the unique hardware identifier for a machine
type Fingerprint struct {
	MachineID       string
	ProductUUID     string
	ProductSerial   string
	BoardSerial     string
	ChassisSerial   string
	Normalized      string // Final normalized fingerprint string
	Hash            string // SHA256 hash of the normalized fingerprint
}

// SourceInfo tracks which file a value came from
type SourceInfo struct {
	Path   string
	Value  string
	Exists bool
}

// CollectAndHash reads all available hardware identifiers and returns a SHA256 hash
func CollectAndHash() (string, error) {
	fp, err := Collect()
	if err != nil {
		return "", err
	}
	return fp.Hash, nil
}

// Collect reads all available hardware identifiers from the system
func Collect() (*Fingerprint, error) {
	fingerprint := &Fingerprint{}

	// Read machine-id (primary source)
	fingerprint.MachineID = readFileTrim("/etc/machine-id")
	if fingerprint.MachineID == "" {
		// Fallback to dbus machine-id
		fingerprint.MachineID = readFileTrim("/var/lib/dbus/machine-id")
	}

	// Read DMI identifiers (most reliable on VMs/servers)
	fingerprint.ProductUUID = readDMIFile("product_uuid")
	if fingerprint.ProductUUID == "" {
		fingerprint.ProductUUID = readDMIFile("product_serial")
	}
	if fingerprint.ProductUUID == "" {
		fingerprint.ProductUUID = readDMIFile("board_serial")
	}
	if fingerprint.ProductUUID == "" {
		fingerprint.ProductUUID = readDMIFile("chassis_serial")
	}

	// Build normalized string
	fingerprint.buildNormalized()

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(fingerprint.Normalized))
	fingerprint.Hash = hex.EncodeToString(hash[:])

	return fingerprint, nil
}

// buildNormalized creates a deterministic string representation of the fingerprint
func (fp *Fingerprint) buildNormalized() {
	var parts []string

	// Add machine-id with source prefix for traceability
	if fp.MachineID != "" {
		parts = append(parts, fmt.Sprintf("machine-id:%s", strings.TrimSpace(fp.MachineID)))
	}

	// Add DMI identifiers
	if fp.ProductUUID != "" {
		parts = append(parts, fmt.Sprintf("product-uuid:%s", strings.TrimSpace(fp.ProductUUID)))
	}
	if fp.ProductSerial != "" {
		parts = append(parts, fmt.Sprintf("product-serial:%s", strings.TrimSpace(fp.ProductSerial)))
	}
	if fp.BoardSerial != "" {
		parts = append(parts, fmt.Sprintf("board-serial:%s", strings.TrimSpace(fp.BoardSerial)))
	}
	if fp.ChassisSerial != "" {
		parts = append(parts, fmt.Sprintf("chassis-serial:%s", strings.TrimSpace(fp.ChassisSerial)))
	}

	// Sort deterministically for reproducibility
	sort.Strings(parts)

	// Join with newline separator
	fp.Normalized = strings.Join(parts, "\n")
}

// readFileTrim reads a file and returns its trimmed content
func readFileTrim(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// readDMIFile reads a DMI file from /sys/class/dmi/id/
func readDMIFile(filename string) string {
	path := fmt.Sprintf("/sys/class/dmi/id/%s", filename)
	return readFileTrim(path)
}

// GetHash returns the SHA256 hash of the fingerprint
func (fp *Fingerprint) GetHash() string {
	return fp.Hash
}

// String returns a human-readable representation
func (fp *Fingerprint) String() string {
	return fmt.Sprintf("Fingerprint: %s\nHash: %s", fp.Normalized, fp.Hash)
}
