# License Verification

This directory contains the license and authorization logic for `harden-sles15`.

## Files

- `license.go` - Main license verification code (whitelist, key derivation)
- `keys/dev.pem.example` - Example development private key template

## Workflow

1. **Fingerprint Collection**: Run `fingerprint-collector.bin` on target device
2. **Authorization**: Customer submits fingerprint to `support@yourorg.com`
3. **Whitelist Update**: Admin adds hash to `authorizedFingerprints` map in `license.go`
4. **Build & Deploy**: Rebuild binary with updated whitelist

## Security Notes

- Never commit actual private keys (`.pem`, `.key`) - they're gitignored
- Use `license/keys/dev.pem.example` as template for key generation
- In production, consider loading whitelist from an encrypted external store
- Device-specific encryption keys are derived using HMAC-SHA256(fingerprint, master_secret)
