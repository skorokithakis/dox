# Dox - Container Command Wrapper

Dox is a lightweight wrapper utility that transparently executes commands within Docker or Podman containers while maintaining the user experience of native host commands. It eliminates the need to install tools directly on the host system by running them in isolated, reproducible container environments.

## Features

- **Transparent Execution**: Run containerized commands as if they were native
- **Signal Forwarding**: Full support for SIGINT, SIGTERM, and other signals
- **File System Integration**: Automatic mounting of current directory to `/workspace`
- **Permission Preservation**: Runs with host UID/GID to maintain file ownership
- **Environment Variables**: Pass through host environment variables
- **Docker & Podman Support**: Works with both container runtimes
- **Inline Dockerfiles**: Build custom images on-the-fly
- **Image Management**: Upgrade and clean commands for maintenance

## Installation

### From Source

```bash
git clone https://github.com/skorokithakis/dox.git
cd dox
make build
sudo cp build/dox /usr/local/bin/
```

### Prerequisites

- Go 1.21 or later (for building from source)
- Docker or Podman installed and running
- XDG Base Directory support (Linux/macOS)

## Quick Start

1. Create a configuration directory:
```bash
mkdir -p ~/.config/dox/commands
```

2. Create a command configuration (e.g., for Python):
```bash
cat > ~/.config/dox/commands/python.yaml << EOF
image: python:3.11-slim
volumes:
  - .:/workspace
  - \${HOME}/.cache/pip:/root/.cache/pip
environment:
  - PYTHONPATH
  - VIRTUAL_ENV
EOF
```

3. Run Python through dox:
```bash
dox run python script.py
dox run python -c "print('Hello from container!')"
```

## Configuration

### Global Configuration

Create `~/.config/dox/config.yaml`:

```yaml
runtime: docker  # or "podman"
```

### Command Configuration

Command configurations are stored in `~/.config/dox/commands/<command>.yaml`:

```yaml
image: node:20-alpine       # Required: Docker image to use
volumes:                     # Optional: Additional volume mounts
  - ${HOME}/.npm:/root/.npm
  - ${HOME}/.yarn:/root/.yarn
environment:                 # Optional: Environment variables to pass
  - NODE_ENV
  - NPM_TOKEN
command: node               # Optional: Override the command
```

### Inline Dockerfile Example

For custom images, use inline Dockerfiles:

```yaml
image: mycommand:latest
build:
  dockerfile_inline: |
    FROM ubuntu:22.04
    RUN apt-get update && apt-get install -y curl git vim
    WORKDIR /workspace
    ENTRYPOINT ["bash"]
volumes:
  - .:/workspace
```

## Built-in Commands

```bash
dox list                 # List available commands
dox version              # Show dox version
dox upgrade <command>    # Upgrade a command's image
dox upgrade-all          # Upgrade all images
dox clean                # Remove stopped containers
```

## Examples

### Python Development

```yaml
# ~/.config/dox/commands/python.yaml
image: python:3.11-slim
volumes:
  - .:/workspace
  - ${HOME}/.cache/pip:/root/.cache/pip
environment:
  - PYTHONPATH
  - VIRTUAL_ENV
```

Usage:
```bash
dox run python script.py
dox run python -m pip install requests
dox run python -c "import sys; print(sys.version)"
```

### Node.js with Custom Build

```yaml
# ~/.config/dox/commands/node.yaml
image: node:custom
build:
  dockerfile_inline: |
    FROM node:20-alpine
    RUN npm install -g yarn pnpm typescript
    WORKDIR /workspace
volumes:
  - .:/workspace
  - ${HOME}/.npm:/root/.npm
environment:
  - NODE_ENV
  - NPM_TOKEN
```

Usage:
```bash
dox run node index.js
dox run node -e "console.log(process.version)"
```

### Go Development

```yaml
# ~/.config/dox/commands/go.yaml
image: golang:1.21
volumes:
  - .:/workspace
  - ${HOME}/go:/go
  - ${HOME}/.cache/go-build:/root/.cache/go-build
environment:
  - GOPROXY
  - GOPRIVATE
  - CGO_ENABLED
```

Usage:
```bash
dox run go build ./...
dox run go test -v ./...
dox run go mod tidy
```

## Advanced Features

### Volume Mounts

- Current directory is always mounted to `/workspace`
- Additional volumes can be specified in configuration
- Environment variables are expanded: `${HOME}`, `${XDG_CONFIG_HOME}`
- Read-only mounts supported: `/host/path:/container/path:ro`

### Signal Handling

Dox forwards all signals to the containerized process:
```bash
dox run sleep 30  # Can be interrupted with Ctrl+C
```

### Concurrent Execution

Multiple instances of the same command can run simultaneously:
```bash
dox run python server.py &
dox run python client.py
```

## Troubleshooting

### Docker Daemon Not Running

```
Error: Docker daemon not responding. Is Docker running?
```

Solution: Start Docker or Podman service:
```bash
sudo systemctl start docker
# or
systemctl --user start podman
```

### Permission Denied

Dox runs containers with your host UID/GID. If you encounter permission issues:
1. Ensure the image supports running as non-root
2. Check volume mount permissions
3. Consider using a custom Dockerfile with proper user setup

### Command Not Found

```
Error: command 'xyz' doesn't exist. Create ~/.config/dox/commands/xyz.yaml
```

Solution: Create the configuration file for the command.

## Architecture

Dox works by:
1. Parsing command-line arguments
2. Loading configuration for the requested command
3. Creating a container with appropriate settings
4. Forwarding signals and I/O streams
5. Returning the container's exit code

Key design decisions:
- Single static binary for easy distribution
- XDG Base Directory compliance
- Transparent I/O handling
- Host network mode for simplicity
- Automatic cleanup with `--rm` flag

## Development

### Building

```bash
make build        # Build binary
make test         # Run tests
make install      # Install to GOPATH/bin
make dev          # Build with race detector
```

### Project Structure

```
dox/
├── cmd/dox/          # Entry point
├── internal/
│   ├── cli/          # Command-line interface
│   ├── config/       # Configuration management
│   ├── runtime/      # Docker/Podman abstraction
│   └── utils/        # Utilities
├── go.mod            # Go modules
└── Makefile          # Build automation
```

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

MIT License - See LICENSE file for details

## Acknowledgments

Based on the RFC specification for containerized command execution. Inspired by the need for clean, reproducible development environments without system pollution.