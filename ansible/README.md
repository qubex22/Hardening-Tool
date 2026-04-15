# Ansible Playbook Structure

This directory contains the Ansible playbook and roles for SLES 15 hardening.

## Files

- `playbook.yml` - Top-level playbook that includes sles15_cis role
- `roles/sles15_cis/` - CIS benchmark-based hardening role

### Role Structure

```
sles15_cis/
├── tasks/main.yml      # Main hardening tasks (CIS benchmarks)
├── handlers/main.yml   # Ansible handlers for service management
├── defaults/main.yml   # Default variables
└── templates/          # Jinja2 templates for config files
```

## Usage

The playbook is embedded in the binary and executed by `main.go`.

To test locally (without embedded binary):

```bash
# Set hardening level: 0=disabled, 1=basic, 2=medium, 3=maximum
export HARDENING_LEVEL=1

# Run with local connection
ansible-playbook -c local playbook.yml

# Or run via the binary
sudo ./harden-sles15.bin
```

## Hardening Levels

| Level | Description |
|-------|-------------|
| 0 | No hardening (disabled) |
| 1 | Basic security (filesystem, user config) |
| 2 | Medium security (password policy, audit) |
| 3 | Maximum security (full CIS compliance) |
