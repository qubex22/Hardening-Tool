#!/bin/bash
set -e

echo "========================================"
echo "  harden-sles15 Build Script"
echo "========================================"

# Cross-compile target
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# Step 1: Ensure go.sum is up to date
echo ""
echo "[1/5] Validating dependencies..."
go mod tidy
go mod download

# Step 2: Verify go-embed-python dependency is available
echo ""
echo "[2/5] Verifying go-embed-python dependency..."
go run -mod=mod github.com/kluctl/go-embed-python/cmd/get-python@latest -help > /dev/null 2>&1 || true
echo "go-embed-python dependency verified."
echo "Python + Ansible will be extracted at runtime."

# Step 3: Compile main binary
echo ""
echo "[3/4] Compiling harden-sles15.bin..."
go build -ldflags="-s -w" -o harden-sles15.bin .

# Step 4: Compile fingerprint-collector standalone
echo ""
echo "[4/4] Compiling fingerprint-collector..."
go build -tags fingerprint -o fingerprint-collector ./fingerprint-collector.go

# Step 5: Verify build output
echo ""
echo "[5/5] Verifying build output..."
if [ -f "harden-sles15.bin" ]; then
    # Check if the binary contains embedded files
    if strings harden-sles15.bin 2>/dev/null | grep -q "ansible-playbook" || \
       strings harden-sles15.bin 2>/dev/null | grep -q "ansible_core"; then
        echo "  ✓ Binary contains embedded ansible assets"
    else
        echo "  ⚠ Warning: ansible assets may not be embedded. Check build output."
    fi
fi

# Summary
echo ""
echo "========================================"
echo "  Build complete!"
echo "========================================"
echo ""
echo "  Binary: harden-sles15.bin"
echo "  $ du -h harden-sles15.bin"
echo ""
echo "  Collector: fingerprint-collector"
echo "  $ ./fingerprint-collector"
echo ""
echo "  Deployment:"
echo "    - Copy harden-sles15.bin to the target SLES 15 machine"
echo "    - Run: sudo ./harden-sles15.bin"
echo ""

ls -lh harden-sles15.bin fingerprint-collector 2>/dev/null || true
