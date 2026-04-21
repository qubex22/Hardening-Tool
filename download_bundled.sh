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

# Download ansible-core and all dependencies as wheels
pip wheel --wheel-dir "$TMPDIR_BUILD/wheels" --no-cache-dir ansible-core

# Download ansible.posix collection
ansible-galaxy collection install ansible.posix --roles-path "$TMPDIR_BUILD/galaxy" 2>/dev/null || true

# Package the collection if galaxy downloaded it
if [ -d "$TMPDIR_BUILD/galaxy/ansible_collections/ansible/posix" ]; then
    mkdir -p "$TMPDIR_BUILD/galaxy/tarball"
    cp -r "$TMPDIR_BUILD/galaxy/ansible_collections" "$TMPDIR_BUILD/galaxy/tarball/"
    (
        cd "$TMPDIR_BUILD/galaxy/tarball"
        tar -czf "$TMPDIR_BUILD/ansible-posix.tar.gz" ansible_collections/
    )
    cp "$TMPDIR_BUILD/ansible-posix.tar.gz" "$BUNDLE_DIR/"
    echo "ansible.posix collection bundled."
else
    echo "WARNING: ansible.posix collection download failed."
fi

# Copy all wheels to bundled directory
cp "$TMPDIR_BUILD/wheels"/* "$BUNDLE_DIR/" 2>/dev/null || true

echo "Bundled packages:"
ls -lh "$BUNDLE_DIR/"
