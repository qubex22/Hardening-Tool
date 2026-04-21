//go:build fingerprint
// +build fingerprint

// fingerprint-collector: A standalone tool to collect device fingerprints for authorization
//
// Usage:
//   go run fingerprint-collector.go > fingerprint.txt
//   ./fingerprint-collector > fingerprint.txt
//
// Output format:
//   sha256:<hash>
package main

import (
	"fmt"
	"os"

	"harden-sles15/fingerprint"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("fingerprint-collector - Collect device fingerprint for authorization")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ./fingerprint-collector [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --help     Show this help message")
		fmt.Println("  --json     Output in JSON format")
		fmt.Println("")
		fmt.Println("Output:")
		fmt.Println("  A SHA256 hash of the device fingerprint.")
		fmt.Println("  Submit this to your organization's support team for authorization.")
		os.Exit(0)
	}

	fp, err := fingerprint.Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting fingerprint: %v\n", err)
		os.Exit(1)
	}

	outputFormat := "plain" // default
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		outputFormat = "json"
	}

	switch outputFormat {
	case "json":
		fmt.Printf(`{
  "fingerprint": %q,
  "hash": %q,
  "machine_id": %q,
  "product_uuid": %q
}
`, fp.Normalized, fp.Hash, fp.MachineID, fp.ProductUUID)
	default:
		fmt.Printf("sha256:%s\n", fp.GetHash())
	}

	fmt.Fprintf(os.Stderr, "\n# Device fingerprint for authorization\n")
	fmt.Fprintf(os.Stderr, "# Submit this hash to your organization's support team\n")
	fmt.Fprintf(os.Stderr, "# Hash: %s\n", fp.GetHash())
}
