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

# Step 2: Generate bundled pip packages (needs internet)
echo ""
echo "[2/5] Generating bundled Python packages..."
cd python/pip_gen
go mod tidy
go run main.go
cd ../..
echo "Bundled packages generated."

# Step 3: Download and bundle ansible collections (needs internet)
echo ""
echo "[3/5] Downloading ansible collections..."
TMP_GALAXY=$(mktemp -d)

# Download ansible.posix collection using ansible-galaxy
python3 -m ansible.galaxy install ansible.posix --collections-path "$TMP_GALAXY" 2>/dev/null || {
    # Fallback: download tarball directly from GitHub
    echo "ansible-galaxy not available, downloading from GitHub..."
    COLLECTION_VERSION="1.5.6"
    curl -sL "https://github.com/ansible-collections/ansible.posix/archive/refs/tags/${COLLECTION_VERSION}.tar.gz" \
        -o "python/bundled/ansible-posix-${COLLECTION_VERSION}.tar.gz"
    rm -rf "$TMP_GALAXY"
    exit 0
}

# If galaxy installed it as a directory, package it as a tarball
if [ -d "$TMP_GALAXY/ansible_collections/ansible/posix" ]; then
    mkdir -p "$TMP_GALAXY/tarball"
    cp -r "$TMP_GALAXY/ansible_collections" "$TMP_GALAXY/tarball/"
    cd "$TMP_GALAXY/tarball"
    tar -czf "$PWD/../../python/bundled/ansible-posix.tar.gz" ansible_collections/
    cd -
fi

rm -rf "$TMP_GALAXY"
echo "Collections downloaded and bundled."

# Step 4: Compile main binary
echo ""
echo "[4/5] Compiling harden-sles15.bin..."
go build -ldflags="-s -w" -o harden-sles15.bin .

# Step 5: Compile fingerprint-collector standalone
echo ""
echo "[5/5] Compiling fingerprint-collector..."
go build -tags fingerprint -o fingerprint-collector ./fingerprint-collector.go

# Verify build output
echo ""
echo "Verifying build output..."
if [ -f "harden-sles15.bin" ]; then
    if strings harden-sles15.bin 2>/dev/null | grep -q "ansible-playbook" || \
       strings harden-sles15.bin 2>/dev/null | grep -q "ansible_core"; then
        echo "  ✓ Binary contains embedded ansible assets"
    else
        echo "  ⚠ Warning: ansible assets may not be embedded."
    fi
    BUNDLE_SIZE=$(du -sh python/bundled 2>/dev/null | cut -f1 || echo "N/A")
    echo "  Bundled packages size: ${BUNDLE_SIZE}"
fi

# Summary
echo ""
echo "========================================"
echo "  Build complete!"
echo "========================================"
echo ""
echo "  Binary: harden-sles15.bin"
echo ""
echo "  Collector: fingerprint-collector"
echo ""
echo "  Deployment:"
echo "    - Copy harden-sles15.bin to the target SLES 15 machine"
echo "    - Run: sudo ./harden-sles15.bin"
echo ""

ls -lh harden-sles15.bin fingerprint-collector 2>/dev/null || true
