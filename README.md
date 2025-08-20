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

## Changelog

See [the changelog](./CHANGELOG.md).

## License

[MIT © Sasha Koss](https://koss.nocorp.me/mit/)
