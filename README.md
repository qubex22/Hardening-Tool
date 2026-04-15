# harden-sles15 — SLES 15 Security Hardening Tool

A standalone, self-contained binary for applying CIS benchmarks and security hardening to SUSE Linux Enterprise Server 15 (SP0-SP4).

## Overview

`harden-sles15` is an enterprise-grade hardening tool that:

- ✅ Runs **without dependencies** on the target host (no Python needed)
- 🔒 Uses **hardware-based authorization** via device fingerprinting
- 📦 Embeds full Python runtime and Ansible playbook
- 🛡️ Protects IP with layered security (binary + optional encryption)

## Target Environment

| Requirement | Value |
|-------------|-------|
| OS | SLES 15 SP0–SP4 |
| Architecture | x86_64 only |
| Privileges | Requires `sudo` |
| Internet | Not required |

## Quick Start

### Build the Binary

```bash
# Make build script executable and run
chmod +x build.sh
./build.sh
```

This produces `harden-sles15.bin` (~200–500 MB).

### Run on Target Device

```bash
sudo ./harden-sles15.bin
```

## Fingerprint Authorization

To authorize a device:

1. Collect fingerprint (on target device):
   ```bash
   ./fingerprint-collector > customer-fingerprint.txt
   ```

2. Submit `customer-fingerprint.txt` to your organization's support team

3. Add the hash to `license/authorizedFingerprints` in `license/license.go`

4. Rebuild the binary with updated whitelist

### Fingerprint Format

```
sha256:1a3b4c...9z
```

## Project Structure

```
harden-sles15/
├── main.go                 # Entry point: orchestrates fingerprint → license → execute
├── fingerprint-collector.go # Standalone fingerprint collection utility
├── build.sh                # Cross-compile script for Linux/amd64
├── Dockerfile.build        # Build environment (SLES 15 + Go)
├── go.mod                  # Module dependencies
│
├── fingerprint/            # Device identification logic
│   ├── fingerprint.go      # Collects machine-id, DMI info
│   └── fingerprint_test.go # Unit tests
│
├── license/                # Authorization & IP protection
│   ├── license.go          # Whitelist check + key derivation
│   ├── keys/               # Private keys (NOT committed)
│   └── README.md           # License workflow documentation
│
├── ansible/                # Embedded playbook and roles
│   ├── playbook.yml        # Top-level playbook
│   └── roles/sles15_cis/   # CIS hardening role
│
├── python/                 # Embedded Python runtime (via go-embed-python)
│   ├── py_embed.go         # Python initialization
│   └── runtime.go          # Runtime management
│
├── ansible_runner/         # Go wrapper around apenella/go-ansible
│   ├── runner.go           # Playbook execution logic
│   └── runner_test.go      # Tests
│
└── tests/
    ├── sles15_test_vm/     # CI-friendly VM setup instructions
    └── test_fingerprint.sh # Fingerprint validation script
```

## Build Requirements

| Tool | Version |
|------|---------|
| Go | 1.21+ |
| Python | 3.11 (for embedded runtime) |
| OpenSSL | For key generation |

## Security Model

### Layer 1: Device Fingerprinting
Collects hardware identifiers from:
- `/etc/machine-id` (primary)
- `/sys/class/dmi/id/*` (fallback)

Hashed with SHA256 for deterministic output.

### Layer 2: Whitelist Check
On startup, the binary validates the device fingerprint against an embedded whitelist. Unauthorized devices exit with code 3 and a clear error message.

### Layer 3 (Optional): Playbook Encryption
Encrypt playbook assets with AES-256-GCM using a device-specific key derived from the fingerprint:
```
Key = HMAC-SHA256(fingerprint, master_secret_key)
```

## Hardening Levels

| Level | Description |
|-------|-------------|
| 0 | No hardening (disabled) |
| 1 | Basic: filesystem config, user permissions |
| 2 | Medium: password policy, audit logging |
| 3 | Maximum: full CIS compliance |

Set via `HARDENING_LEVEL` environment variable.

## License

This tool is proprietary. See `license/README.md` for authorization workflow.

## Support

Contact: support@yourorg.com

## Contributing

Contributions are not accepted at this time due to IP protection requirements.
