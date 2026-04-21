#!/bin/bash
set -e

# Download ansible-core wheels and ansible.posix collection
# This script runs at Docker build time with internet access
# Output: python/bundled/*.whl, python/bundled/ansible-*.tar.gz

BUNDLE_DIR="$1"
mkdir -p "$BUNDLE_DIR"

TMPDIR_BUILD=$(mktemp -d)
trap "rm -rf $TMPDIR_BUILD" EXIT

python3 -m venv "$TMPDIR_BUILD/venv"
. "$TMPDIR_BUILD/venv/bin/activate"

# Install ansible-core into the venv (so ansible-galaxy is available)
# Then also download wheels for offline embedding
echo "Installing ansible-core into venv..."
pip install --no-cache-dir ansible-core

# Download all dependency wheels for embedding
echo "Downloading wheels..."
pip wheel --wheel-dir "$TMPDIR_BUILD/wheels" --no-cache-dir ansible-core

# Download ansible.posix collection as a pre-built tarball from Galaxy
echo "Downloading ansible.posix collection tarball..."
mkdir -p "$TMPDIR_BUILD/galaxy/downloads"
cd "$TMPDIR_BUILD/galaxy/downloads"

# Try ansible-galaxy download first
if ansible-galaxy collection download ansible.posix -p . 2>/dev/null; then
    echo "Downloaded via ansible-galaxy collection download"
else
    # Fallback: install then package
    echo "Fallback: installing then packaging..."
    mkdir -p "$TMPDIR_BUILD/galaxy/installed"
    ansible-galaxy collection install ansible.posix --collections-path "$TMPDIR_BUILD/galaxy/installed" 2>&1 || {
        echo "ERROR: Failed to download ansible.posix collection"
        exit 1
    }
fi

# Debug: show what we have
echo "Contents of downloads dir:"
ls -laR "$TMPDIR_BUILD/galaxy/downloads/" 2>&1 || true
echo "Contents of installed dir:"
ls -laR "$TMPDIR_BUILD/galaxy/installed/" 2>&1 || true

# Package the collection from whichever source is available
COLLECTION_SRC=""

# Check for pre-built tarball from ansible-galaxy download
if ls "$TMPDIR_BUILD/galaxy/downloads/ansible-posix-"*.tar.gz 1>/dev/null 2>&1; then
    cp "$TMPDIR_BUILD/galaxy/downloads/ansible-posix-"*.tar.gz "$BUNDLE_DIR/ansible-posix.tar.gz"
    echo "Pre-built ansible.posix tarball bundled."
    COLLECTION_SRC="prebuilt"
fi

# Check for installed collection
if [ -z "$COLLECTION_SRC" ] && [ -d "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix" ]; then
    mkdir -p "$TMPDIR_BUILD/collection_pkg/posix"
    cp -r "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix/"* "$TMPDIR_BUILD/collection_pkg/posix/"
    (
        cd "$TMPDIR_BUILD/collection_pkg"
        tar -czf "$TMPDIR_BUILD/ansible-posix.tar.gz" posix/
    )
    cp "$TMPDIR_BUILD/ansible-posix.tar.gz" "$BUNDLE_DIR/"
    echo "ansible.posix collection bundled from installed source."
    COLLECTION_SRC="installed"
fi

# Check for installed collection in alternative structure
if [ -z "$COLLECTION_SRC" ] && [ -d "$TMPDIR_BUILD/galaxy/installed/ansible_collections/ansible/posix" ]; then
    mkdir -p "$TMPDIR_BUILD/collection_pkg/posix"
    cp -r "$TMPDIR_BUILD/galaxy/installed/ansible_collections/ansible/posix/"* "$TMPDIR_BUILD/collection_pkg/posix/"
    (
        cd "$TMPDIR_BUILD/collection_pkg"
        tar -czf "$TMPDIR_BUILD/ansible-posix.tar.gz" posix/
    )
    cp "$TMPDIR_BUILD/ansible-posix.tar.gz" "$BUNDLE_DIR/"
    echo "ansible.posix collection bundled from installed source."
    COLLECTION_SRC="installed"
fi

if [ -z "$COLLECTION_SRC" ]; then
    echo "ERROR: Could not find ansible.posix collection anywhere"
    echo "Searched:"
    find "$TMPDIR_BUILD/galaxy" -name "*.tar.gz" -o -name "MANIFEST.json" 2>/dev/null
    exit 1
fi

# Copy all wheels to bundled directory
cp "$TMPDIR_BUILD/wheels"/* "$BUNDLE_DIR/" 2>/dev/null || true

echo "Bundled packages:"
ls -lh "$BUNDLE_DIR/"
