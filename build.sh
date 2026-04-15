#!/bin/bash
set -e

export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0  # Static binary (no glibc dependency)

# Build embedded Python assets first
echo "Building embedded Python assets..."
go run ./python/py_embed.go --dump-assets ./python/assets

# Compile main binary (strip debug symbols)
echo "Compiling harden-sles15.bin..."
go build -ldflags="-s -w -extldflags '-static'" \
    -o harden-sles15.bin .

echo ""
echo "========================================"
echo "Build complete!"
ls -lh harden-sles15.bin
echo "========================================"
