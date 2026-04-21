#!/bin/bash
set -e

echo "========================================"
echo "  harden-sles15 Build Script"
echo "========================================"

# Cross-compile target
export GOOS=linux
export GOARCH=amd64
# CGO_ENABLED=1 is required for go-embed-python to link properly
export CGO_ENABLED=1

# Step 1: Ensure go.sum is up to date
echo ""
echo "[1/4] Validating dependencies..."
go mod tidy
go mod download

# Step 2: Build embedded Python assets
echo ""
echo "[2/4] Building embedded Python assets..."
if [ ! -d "./python/assets" ] || [ -z "$(ls -A ./python/assets 2>/dev/null)" ]; then
    go run ./python/py_embed.go --dump-assets ./python/assets
    echo "Python assets extracted to ./python/assets/"
else
    echo "Python assets already exist, skipping..."
fi

# Step 3: Compile main binary
echo ""
echo "[3/4] Compiling harden-sles15.bin..."
go build -ldflags="-s -w" -o harden-sles15.bin .

# Step 4: Compile fingerprint-collector standalone
echo ""
echo "[4/4] Compiling fingerprint-collector..."
go build -o fingerprint-collector ./fingerprint-collector.go

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

ls -lh harden-sles15.bin fingerprint-collector
