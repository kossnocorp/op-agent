# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Building

- `just build` - Build both server and client binaries for current platform
- `just build-server` - Build only the op-agent server
- `just build-client` - Build only the op-agent-client
- `just build-all` - Build for all supported platforms (Linux, macOS, Windows)
- `just clean` - Remove all built binaries

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

### Version Management

The version information is embedded from `VERSION` file using Go embed directive in [`./version.go`](./version.go).
