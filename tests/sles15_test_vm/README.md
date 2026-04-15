# SLES 15 Test VM Setup

This directory contains configuration for testing the hardener in a QEMU/KVM VM.

## Requirements

- QEMU/KVM
- libvirt
- cloud-init

## Quick Start

1. Download SLES 15 SPx cloud image:
   ```bash
   wget https://download.suse.com/.../SLES-15-SPX.qcow2
   ```

2. Create VM from image with cloud-init:
   ```yaml
   # cloud-init.yml
   #cloud-config
   chpasswd:
     list: |
       root:admin123
     expire: false
   ssh_authorized_keys:
     - | 
       $(cat ~/.ssh/id_rsa.pub)
   ```

3. Run with:
   ```bash
   qemu-system-x86_64 \
     -enable-kvm \
     -m 4096 \
     -drive file=SLES-15-SPX.qcow2,format=qcow2 \
     -cdrom cloud-init.iso \
     -netdev user,id=net0,hostfwd=tcp::2222-:22 \
     -device virtio-net-pci,netdev=net0
   ```

## CI Testing

For CI (GitHub Actions/GitLab CI), use:

```yaml
name: SLES 15 Hardening Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run tests in SLES 15 VM
        run: |
          docker build -f Dockerfile.test -t hardener-test .
          docker run hardener-test
```

## Test Checklist

- [ ] Binary runs without Python installed on host
- [ ] Fingerprint collection works
- [ ] License validation passes for whitelisted device
- [ ] Playbook executes successfully
- [ ] Hardening changes are applied (verify with `cat /etc/passwd`, etc.)
