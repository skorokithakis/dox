# Dox example configurations

This directory contains sample configuration files for various commands that can be run through dox.

## Setup

Copy these files to your dox configuration directory:

```bash
mkdir -p ~/.config/dox/commands
cp claude.yaml ~/.config/dox/commands/
cp ls.yaml ~/.config/dox/commands/
```

## Usage examples

### Claude Code

Once configured, you can run Claude Code in a container:

```bash
# Interactive mode
dox run claude

# With specific command
dox run claude "help me write a function to calculate fibonacci numbers"

# Use claude with current directory mounted
dox run claude "analyze the code in this directory"

# Resume a previous session
dox run claude --resume
```

### ls command

A simple ls replacement that runs in a minimal container:

```bash
# List files in current directory
dox run ls

# List with options
dox run ls -la

# List specific directory
dox run ls /workspace/src

# List with human-readable sizes
dox run ls -lh
```

## Advanced examples

### Python with specific version

```yaml
# ~/.config/dox/commands/python3.12.yaml
image: python:3.12-slim
volumes:
  - .:/workspace
  - ${HOME}/.cache/pip:/root/.cache/pip
environment:
  - PYTHONPATH
```

Usage:
```bash
dox run python3.12 -m venv .venv
dox run python3.12 script.py
```

### Rust development

```yaml
# ~/.config/dox/commands/cargo.yaml
image: rust:latest
volumes:
  - .:/workspace
  - ${HOME}/.cargo:/root/.cargo
environment:
  - CARGO_HOME
  - RUSTUP_HOME
```

Usage:
```bash
dox run cargo build --release
dox run cargo test
dox run cargo clippy
```

### AWS CLI

```yaml
# ~/.config/dox/commands/aws.yaml
image: amazon/aws-cli:latest
volumes:
  - .:/workspace
  - ${HOME}/.aws:/root/.aws:ro
environment:
  - AWS_PROFILE
  - AWS_REGION
  - AWS_ACCESS_KEY_ID
  - AWS_SECRET_ACCESS_KEY
  - AWS_SESSION_TOKEN
command: aws
```

Usage:
```bash
dox run aws s3 ls
dox run aws ec2 describe-instances
dox run aws lambda list-functions
```

### Terraform

```yaml
# ~/.config/dox/commands/terraform.yaml
image: hashicorp/terraform:latest
volumes:
  - .:/workspace
  - ${HOME}/.terraform.d:/root/.terraform.d
environment:
  - TF_LOG
  - TF_VAR_region
```

Usage:
```bash
dox run terraform init
dox run terraform plan
dox run terraform apply
```

## Tips

1. **Caching**: Mount cache directories to speed up repeated operations (pip, npm, cargo, etc.)
2. **Security**: Use read-only mounts (`:ro`) for sensitive files like SSH keys
3. **Environment**: Pass through only necessary environment variables to maintain isolation
4. **Custom images**: Use inline Dockerfiles for tools that need specific configurations
5. **Versions**: Pin image tags for reproducibility (e.g., `python:3.11.7` instead of `python:3.11`)