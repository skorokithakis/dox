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
dox claude

# With specific command
dox claude "help me write a function to calculate fibonacci numbers"

# Use claude with current directory mounted
dox claude "analyze the code in this directory"

# Resume a previous session
dox claude --resume
```

### ls command

A simple ls replacement that runs in a minimal container:

```bash
# List files in current directory
dox ls

# List with options
dox ls -la

# List specific directory
dox ls /workspace/src

# List with human-readable sizes
dox ls -lh
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
dox python3.12 -m venv .venv
dox python3.12 script.py
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
dox cargo build --release
dox cargo test
dox cargo clippy
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
dox aws s3 ls
dox aws ec2 describe-instances
dox aws lambda list-functions
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
dox terraform init
dox terraform plan
dox terraform apply
```

## Tips

1. **Caching**: Mount cache directories to speed up repeated operations (pip, npm, cargo, etc.)
2. **Security**: Use read-only mounts (`:ro`) for sensitive files like SSH keys
3. **Environment**: Pass through only necessary environment variables to maintain isolation
4. **Custom images**: Use inline Dockerfiles for tools that need specific configurations
5. **Versions**: Pin image tags for reproducibility (e.g., `python:3.11.7` instead of `python:3.11`)