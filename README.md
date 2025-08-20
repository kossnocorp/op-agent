# op-agent

`op-agent` allows you to connect to the host 1Password CLI and utilize biometric authentication from remote environments, i.e., [Dev Containers](https://containers.dev/).

It proxies 1Password CLI commands over HTTP, allowing you to request secrets without exposing the [1Password service account](https://developer.1password.com/docs/service-accounts/).

It consists of two binaries: a server (`op-agent`) that runs on the host and a client (`op-agent-client`) for making requests.

## Installation

[Download the `op-agent` and `op-agent-client`](https://github.com/kossnocorp/op-agent/releases/latest) binaries for your platform from the releases page, or build from source:

```sh
go build -o ./dist/op-agent ./cmd/op-agent
go build -o ./dist/op-agent-client ./cmd/op-agent-client
```

## Usage

Start the `op-agent` server on your host machine:

```sh
# Start server at localhost:25519
op-agent start

# Start in non-interactive mode (only pre-approved commands)
op-agent start --non-interactive

# Start in insecure mode (UNSAFE - allows all commands)
op-agent start --insecure

# Print version number (useful for automation and Ansible playbooks)
op-agent --version
```

### Pre-Approve

You can pre-approve commands, which is useful in the non-interactive mode:

```sh
op-agent approve op item get "AWS Token" --vault "Private" --format json
```

### Port

By default, both the `op-agent` server and `op-agent-client` assume the default port `25519`. If it's not available or you want to use a different port, you can set the `OP_AGENT_PORT` environment variable:

```sh
# On the host
OP_AGENT_PORT=4096 op-agent

# In a container
OP_AGENT_PORT=4096 op-agent-client op whoami
```

### Host

`op-agent-client` tries to detect the appropriate host. If `/.dockerenv` or `/run/.containerenv` are present, it will use `host.docker.internal` and fall back to `localhost` otherwise.

You can also customize the port using `OP_AGENT_HOST`:

```sh
# On the host
OP_AGENT_HOST=192.168.1.100 op-agent

# In a container
OP_AGENT_HOST=192.168.1.100 op-agent-client op whoami
```

## Security

### Binary Verification

All release binaries are cryptographically signed using [GitHub's artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations). You can verify that any binary was built from the exact source code in this repository:

```sh
gh attestation verify <binary> --owner kossnocorp
```

This provides tamper-proof assurance that the binary hasn't been modified since it was built from the tagged source code.

### Command Approval

By default, `op-agent` implements a command approval system for enhanced security. When a new 1Password CLI command is requested:

- **Interactive mode** (default): Prompts you to approve each new command with options:

  - `once` - Allow this command once
  - `always` - Allow this command always (saves to config)
  - `no` - Deny the command (default)

- **Non-Interactive mode** (`--non-interactive`): Only allows pre-approved commands from the config file

- **Insecure mode** (`--insecure`): Disables all security checks (**NOT RECOMMENDED**)

Approved commands are stored in `~/.config/op-agent/config.json` on macOS/Linux and `%APPDATA%/op-agent/config.json` on Windows.

All command executions are logged in `~/.local/share/op-agent/commands.log` on macOS/Linux and `%APPDATA%/op-agent/commands.log` on Windows.

### Dependencies

To reduce the vector of the attack, minimize used dependencies (i.e., we don't use more convenient TOML/YAML for the config in favor of vanilla JSON).

To keep dependencies up-to-date, we have GitHub Actions set up to automatically check for updates and create pull requests.

### Why Not 1Password Service Accounts?

[1Password Service Accounts](https://developer.1password.com/docs/service-accounts/) is one of the recommended ways to authenticate 1Password CLI for dev containers and CI/CD. They offer fine-grained scoping and are designed for automated use cases. However, they rely on bearer tokens that can be easily accessed by malicious code running in a remote environment and replayed until you rotate or revoke it.

`op-agent` takes a different approach that is better suited for dev containers. Instead of dropping long-lived tokens into containers, it proxies the 1Password CLI through a local agent that leverages the host's biometric authentication. The 1Password approval expires after 10 minutes, adding an extra security layer to the setup.

This reduces replay risk if a container is compromised, since access is anchored to the developer's presence and device rather than a static credential.

#### When to Use Which

- **Service Accounts**: Headless services, CI/CD, and infrastructure automation that can't depend on human interaction. Accept the operational overhead of scoping and rotating tokens.

- **`op-agent`**: Developer machines and dev containers where tying access to the developer presence is preferable to embedding reusable tokens.

## Changelog

See [the changelog](./CHANGELOG.md).

## License

[MIT © Sasha Koss](https://koss.nocorp.me/mit/)
