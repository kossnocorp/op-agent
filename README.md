# op-agent

`op-agent` allows you to connect to the host 1Password CLI and utilize biometric authentication from remote environments, i.e., [Dev Containers](https://containers.dev/).

It proxies 1Password CLI commands over HTTP, allowing you to request secrets without exposing the [1Password service account](https://developer.1password.com/docs/service-accounts/).

It consists of two binaries: a server (`op-agent`) that runs on the host and a client (`op-agent-client`) for making requests.

## Installation

Download the appropriate binaries for your platform from the releases page, or build from source:

```sh
# Build for current platform
just build

# Build for all platforms
just build-all
```

## Usage

Start the `op-agent` server on your host machine:

```sh
# Start server at localhost:25519
op-agent start
```

Then run `op-agent-client` in a container:

```sh
# Run `op list vaults` on the host and get the response
op-agent op list vaults
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
