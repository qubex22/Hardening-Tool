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
echo "[1/4] Validating dependencies..."
go mod tidy
go mod download

# Step 2: Download and bundle ansible-core wheels and collections (needs internet)
echo ""
echo "[2/4] Downloading and bundling Python packages..."
mkdir -p python/bundled

# Create a temp venv to install ansible and download packages
TMPDIR_BUILD=$(mktemp -d)
python3 -m venv "$TMPDIR_BUILD/venv"
source "$TMPDIR_BUILD/venv/bin/activate"

# Install ansible-core to get all its dependencies as wheels
pip install --wheel-dir "$TMPDIR_BUILD/wheels" --no-cache-dir ansible-core

# Download ansible.posix collection
ansible-galaxy collection install ansible.posix --roles-path "$TMPDIR_BUILD/galaxy" 2>/dev/null || true

# Package the collection if galaxy downloaded it as a directory
if [ -d "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix" ]; then
    mkdir -p "$TMPDIR_BUILD/galaxy/tarball"
    cp -r "$TMPDIR_BUILD/galaxy/ansible_collections" "$TMPDIR_BUILD/galaxy/tarball/"
    cd "$TMPDIR_BUILD/galaxy/tarball"
    tar -czf "$TMPDIR_BUILD/ansible-posix.tar.gz" ansible_collections/
    cd -
    cp "$TMPDIR_BUILD/ansible-posix.tar.gz" python/bundled/
fi

# Copy all wheels to bundled directory
cp "$TMPDIR_BUILD/wheels"/* python/bundled/ 2>/dev/null || true

deactivate
rm -rf "$TMPDIR_BUILD"
echo "Packages bundled."

# Step 3: Compile main binary
echo ""
echo "[3/4] Compiling harden-sles15.bin..."
go build -ldflags="-s -w" -o harden-sles15.bin .

# Step 4: Compile fingerprint-collector standalone
echo ""
echo "[4/4] Compiling fingerprint-collector..."
go build -tags fingerprint -o fingerprint-collector ./fingerprint-collector.go

# Verify build output
echo ""
echo "Verifying build output..."
if [ -f "harden-sles15.bin" ]; then
    if strings harden-sles15.bin 2>/dev/null | grep -q "ansible-playbook" || \
       strings harden-sles15.bin 2>/dev/null | grep -q "ansible_core"; then
        echo "  OK Binary contains embedded ansible assets"
    else
        echo "  WARNING: ansible assets may not be embedded."
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
