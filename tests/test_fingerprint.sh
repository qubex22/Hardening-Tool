#!/bin/bash
# Test script for fingerprint collector

set -e

echo "Testing SLES 15 Hardener Fingerprint Collector"
echo "=============================================="

# Test 1: Check if binary exists and is executable
if [ ! -f "./harden-sles15.bin" ]; then
    echo "[SKIP] Binary not found. Run 'go build' first."
    exit 0
fi

echo ""
echo "[Test 1] Fingerprint collection"
./harden-sles15.bin --fingerprint-only || true

# Test 2: Verify fingerprint format (sha256 hash)
echo ""
echo "[Test 2] Fingerprint format validation"

# Extract just the hash (last part of output)
FP_OUTPUT=$(./harden-sles15.bin --fingerprint-only 2>/dev/null | tail -n1 || true)

if [[ $FP_OUTPUT =~ ^sha256:[a-f0-9]{64}$ ]]; then
    echo "[PASS] Fingerprint format is valid: $FP_OUTPUT"
else
    echo "[FAIL] Invalid fingerprint format: $FP_OUTPUT"
    exit 1
fi

# Test 3: Verify deterministic output (same input = same hash)
echo ""
echo "[Test 3] Deterministic output check"

FP_HASH=$(echo "$FP_OUTPUT" | cut -d: -f2)

for i in {1..3}; do
    NEW_FP=$(./harden-sles15.bin --fingerprint-only 2>/dev/null | tail -n1 | cut -d: -f2 || true)
    if [ "$FP_HASH" != "$NEW_FP" ]; then
        echo "[FAIL] Non-deterministic output detected"
        exit 1
    fi
done

echo "[PASS] Fingerprint is deterministic"

# Test 4: Test on different hardware identifiers
echo ""
echo "[Test 4] Hardware identifier detection"

# Check if we can read machine-id
if [ -f /etc/machine-id ]; then
    echo "[INFO] Found /etc/machine-id"
elif [ -f /var/lib/dbus/machine-id ]; then
    echo "[INFO] Found /var/lib/dbus/machine-id (fallback)"
else
    echo "[WARN] No machine-id found on this system"
fi

# Check DMI info
if [ -d /sys/class/dmi/id ]; then
    echo "[INFO] DMI directory exists"
else
    echo "[WARN] DMI not available on this system"
fi

echo ""
echo "=============================================="
echo "All tests completed!"
echo "=============================================="
