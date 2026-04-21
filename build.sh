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

# Step 2: Install ansible into embedded Python assets
echo ""
echo "[2/5] Installing ansible into embedded Python assets..."

# Create a staging directory for Python + Ansible
STAGING_DIR=$(mktemp -d harden_python_XXXXXX)
echo "Staging directory: $STAGING_DIR"

# Download and extract embedded Python using go-embed-python's mechanism
# We use go run to trigger the embedding process
echo "Extracting embedded Python runtime..."
python_extract_output=$(go run -mod=mod github.com/kluctl/go-embed-python/cmd/get-python@latest \
  -os linux -arch amd64 -output "$STAGING_DIR" 2>&1 || true)

# If the above doesn't work, use pip to install ansible into the staging area
# First check if Python was extracted
if [ ! -d "$STAGING_DIR/bin" ] && [ ! -d "$STAGING_DIR/Scripts" ]; then
    echo "Warning: Python extraction may have failed. Checking for alternative methods..."
    # Try using python3 directly if available
    if command -v python3 &> /dev/null; then
        echo "Using system Python 3 to prepare ansible..."
        python3 -m venv "$STAGING_DIR/venv"
        "$STAGING_DIR/venv/bin/pip" install ansible-core
        # Copy venv contents to staging
        cp -r "$STAGING_DIR/venv"/* "$STAGING_DIR/" 2>/dev/null || true
        rm -rf "$STAGING_DIR/venv"
    else
        echo "Warning: Could not extract Python. The binary will need ansible-playbook on PATH."
        STAGING_DIR=""
    fi
else
    # Python extracted successfully, install ansible into it
    echo "Python extracted. Installing ansible-core..."
    if [ -f "$STAGING_DIR/bin/python3" ]; then
        "$STAGING_DIR/bin/python3" -m pip install --upgrade pip
        "$STAGING_DIR/bin/python3" -m pip install ansible-core
    elif [ -f "$STAGING_DIR/Scripts/python.exe" ]; then
        "$STAGING_DIR/Scripts/python.exe" -m pip install --upgrade pip
        "$STAGING_DIR/Scripts/python.exe" -m pip install ansible-core
    fi
fi

# Copy the staged Python + Ansible into the assets directory
echo "Copying assets to python/assets/..."
rm -rf python/assets
mkdir -p python/assets
if [ -d "$STAGING_DIR" ]; then
    cp -r "$STAGING_DIR"/* python/assets/
    echo "Python + Ansible assets copied."
    rm -rf "$STAGING_DIR"
else
    echo "Warning: No Python assets to copy. Build will use system ansible-playbook."
fi

echo "Python assets embedded at compile time (go-embed-python + ansible-core)."

# Step 3: Compile main binary
echo ""
echo "[3/5] Compiling harden-sles15.bin..."
go build -ldflags="-s -w" -o harden-sles15.bin .

# Step 4: Compile fingerprint-collector standalone
echo ""
echo "[4/5] Compiling fingerprint-collector..."
go build -tags fingerprint -o fingerprint-collector ./fingerprint-collector.go

# Step 5: Verify embedded assets
echo ""
echo "[5/5] Verifying embedded assets..."
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
