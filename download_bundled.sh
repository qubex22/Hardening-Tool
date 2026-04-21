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

# Download ansible.posix collection using the same venv's ansible-galaxy
echo "Downloading ansible.posix collection..."
GALAXY_OUTPUT=$(ansible-galaxy collection install ansible.posix --collections-path "$TMPDIR_BUILD/galaxy" 2>&1) || {
    echo "ansible-galaxy output: $GALAXY_OUTPUT"
    echo "ERROR: Failed to download ansible.posix collection"
    exit 1
}

# Package the collection if galaxy downloaded it
# ansible-galaxy expects tarballs with structure: <namespace>/<collection>/ (containing MANIFEST.json at root)
if [ -d "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix" ]; then
    # Create proper collection tarball structure
    # The tarball root should contain: MANIFEST.json, tasks/, handlers/, etc.
    mkdir -p "$TMPDIR_BUILD/collection_pkg/posix"
    # Copy all contents of the posix collection dir (including MANIFEST.json)
    cp -r "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix/"* "$TMPDIR_BUILD/collection_pkg/posix/"
    (
        cd "$TMPDIR_BUILD/collection_pkg"
        tar -czf "$TMPDIR_BUILD/ansible-posix.tar.gz" posix/
    )
    cp "$TMPDIR_BUILD/ansible-posix.tar.gz" "$BUNDLE_DIR/"
    echo "ansible.posix collection bundled."
else
    echo "ERROR: ansible.posix collection not found at $TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix"
    ls -la "$TMPDIR_BUILD/galaxy/" 2>/dev/null || echo "(galaxy dir does not exist)"
    exit 1
fi

# Copy all wheels to bundled directory
cp "$TMPDIR_BUILD/wheels"/* "$BUNDLE_DIR/" 2>/dev/null || true

echo "Bundled packages:"
ls -lh "$BUNDLE_DIR/"
