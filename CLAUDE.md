# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Building

- `just build` - Build both server and client binaries for current platform
- `just build-server` - Build only the op-agent server
- `just build-client` - Build only the op-agent-client
- `just build-all` - Build for all supported platforms (Linux, macOS, Windows)
- `just clean` - Remove all built binaries

### Server Operation

- `op-agent start` - Start server in interactive mode (prompts for new command approval)
- `op-agent start --non-interactive` - Only allow pre-approved commands from config
- `op-agent start --insecure` - Disable all security checks (NOT RECOMMENDED)

### Testing and Code Quality

- `go test ./...` - Run tests (though no test files currently exist)
- `go fmt ./...` - Format Go code
- `go vet ./...` - Run Go static analysis

## CI/CD

The repository includes GitHub Actions workflows:

- **CI Workflow** (`.github/workflows/ci.yml`) - Runs on push/PR to main branch

  - Runs `go vet`, `go fmt`, `go test`.
  - Builds binaries to ensure compilation works.
  - Triggered on push to main and pull requests.

- **Release Workflow** (`.github/workflows/release.yml`) - Creates releases with artifact attestations.
  - Triggered on git tag push (e.g., `git tag v1.0.0 && git push origin v1.0.0`).
  - Builds binaries for all platforms (Linux, macOS, Windows) in both `amd64` and `arm64`.
  - Generates cryptographic attestations using GitHub's Sigstore integration.
  - Creates GitHub release with attested binaries and checksums.
  - Users can verify binaries with: `gh attestation verify <binary> --owner <owner>`.

## Architecture

This is a client-server system that enables 1Password CLI access from containers by proxying commands through HTTP.

### Core Components

#### Server

[`./cmd/op-agent/main.go`](./cmd/op-agent/main.go).

- HTTP server listening on port `25519` (configurable via `OP_AGENT_PORT`).
- Runs on host machine with access to 1Password CLI and biometric authentication.
- Endpoints:
  - `/op` - Executes 1Password CLI commands. Returns JSON response `OpResponse`.
  - `/handshake` - Version/identity verification endpoint. Returns JSON response `HandshakeResponse`.
- Automatically finds available port if default is occupied.

#### Client

[Source code](./cmd/op-agent-client/main.go).

- Runs inside containers or remote environments.
- Auto-detects container environment via `/.dockerenv` or `/run/.containerenv`.
- Uses `host.docker.internal` for containers, `localhost` otherwise.
- Performs handshake verification before command execution.
- Forwards `op` command arguments to server and streams output.

#### Shared Types

[`./internal/types.go`](./internal/types.go).

- `OpResponse` - Command execution results (stdout, stderr, exit code).
- `HandshakeResponse` - Server identity and version info.

#### Environment Configuration

[`./internal/env.go`](./internal/env.go).

- `OP_AGENT_PORT` - Custom port (default: `25519`).
- `OP_AGENT_HOST` - Custom host address.
- Helper functions for URL construction and environment variable handling.

### Key Design Patterns

- Cobra CLI framework for both binaries.
- JSON over HTTP for communication protocol.
- Container auto-detection for host resolution.
- Version mismatch warnings between client/server.
- Graceful port fallback when default unavailable.
- **Command approval system** for enhanced security - prompts for approval of new commands.
- **Configuration management** using plain JSON (avoiding additional dependencies).
- **Command logging** with timestamps for audit trails.

### Security Features

**Command Approval System**: By default, the server requires approval for new 1Password CLI commands:
- Interactive mode: Prompts user for approval (once/always/no)
- Non-interactive mode: Only allows pre-approved commands from config
- Insecure mode: Disables all checks (use `--insecure` flag)

**Configuration Storage**: 
- Config: `~/.config/op-agent/config.json` (macOS/Linux) or `%APPDATA%/op-agent/config.json` (Windows)
- Logs: `~/.local/share/op-agent/commands.log` (macOS/Linux) or `%APPDATA%/op-agent/commands.log` (Windows)
- Uses plain JSON to minimize attack surface (no additional dependencies)

**Interactive Detection**: Automatically detects CI/CD environments and non-interactive terminals using common environment variables.

### Version Management

The version information is embedded from `VERSION` file using Go embed directive in [`./version.go`](./version.go).
