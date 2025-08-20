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

<details>

<summary>Installing via Ansible playbook</summary>

### Ansible

Here's an example of an Ansible playbook to install `op-agent` and `op-agent-client`:

```yaml
---
- name: Install op-agent and op-agent-client
  hosts: localhost
  connection: local
  vars:
    op_agent_version: "0.2.0"
    op_agent_install_dir: "{{ ansible_env.HOME }}/.local/bin"
    op_agent_arch_map:
      x86_64: amd64
      aarch64: arm64
      armv7l: armv7
    op_agent_os: "{{ ansible_system | lower }}"
    op_agent_arch: "{{ op_agent_arch_map[ansible_architecture] | default(ansible_architecture) }}"
    op_agent_binary_name: "op-agent-{{ op_agent_version }}-{{ op_agent_os }}-{{ op_agent_arch }}"
    op_agent_client_binary_name: "op-agent-client-{{ op_agent_version }}-{{ op_agent_os }}-{{ op_agent_arch }}"
    op_agent_download_url: "https://github.com/kossnocorp/op-agent/releases/download/v{{ op_agent_version }}/{{ op_agent_binary_name }}"
    op_agent_client_download_url: "https://github.com/kossnocorp/op-agent/releases/download/v{{ op_agent_version }}/{{ op_agent_client_binary_name }}"
    op_agent_checksums_url: "https://github.com/kossnocorp/op-agent/releases/download/v{{ op_agent_version }}/checksums.txt"

  tasks:
    - name: Ensure ~/.local/bin directory exists
      file:
        path: "{{ ansible_env.HOME }}/.local/bin"
        state: directory

    - name: Add ~/.local/bin to bash config
      lineinfile:
        path: "{{ ansible_env.HOME }}/.bashrc"
        line: 'export PATH="$HOME/.local/bin:$PATH"'
        create: yes

    - name: Add ~/.local/bin to zsh config
      lineinfile:
        path: "{{ ansible_env.HOME }}/.zshrc"
        line: 'export PATH="$HOME/.local/bin:$PATH"'
        create: yes

    - name: Ensure fish config directory exists
      file:
        path: "{{ ansible_env.HOME }}/.config/fish"
        state: directory

    - name: Add ~/.local/bin to fish config
      lineinfile:
        path: "{{ ansible_env.HOME }}/.config/fish/config.fish"
        line: 'fish_add_path "$HOME/.local/bin"'
        create: yes

    - name: Check if op-agent is installed and get version
      ansible.builtin.command:
        cmd: "{{ op_agent_install_dir }}/op-agent --version"
      register: op_agent_current_version
      changed_when: false
      failed_when: false

    - name: Parse current op-agent version
      ansible.builtin.set_fact:
        current_version: "{{ op_agent_current_version.stdout | trim }}"
      when: op_agent_current_version.rc == 0

    - name: Set current version to 'not installed' if op-agent is not found
      ansible.builtin.set_fact:
        current_version: "not installed"
      when: op_agent_current_version.rc != 0

    - name: Display current and target versions
      ansible.builtin.debug:
        msg:
          - "Current op-agent version: {{ current_version }}"
          - "Target op-agent version: {{ op_agent_version }}"

    - name: Install or update op-agent and op-agent-client if needed
      when: current_version != op_agent_version
      block:
        - name: Create temporary directory for op-agent download
          ansible.builtin.tempfile:
            state: directory
            suffix: op-agent
          register: op_agent_temp_dir

        - name: Download op-agent checksums file
          ansible.builtin.get_url:
            url: "{{ op_agent_checksums_url }}"
            dest: "{{ op_agent_temp_dir.path }}/checksums.txt"
            mode: "0644"

        - name: Download op-agent binary to temp location
          ansible.builtin.get_url:
            url: "{{ op_agent_download_url }}"
            dest: "{{ op_agent_temp_dir.path }}/{{ op_agent_binary_name }}"
            mode: "0755"

        - name: Download op-agent-client binary to temp location
          ansible.builtin.get_url:
            url: "{{ op_agent_client_download_url }}"
            dest: "{{ op_agent_temp_dir.path }}/{{ op_agent_client_binary_name }}"
            mode: "0755"

        - name: Extract expected checksum for op-agent
          ansible.builtin.shell:
            cmd: grep "{{ op_agent_binary_name }}" checksums.txt | awk '{print $1}'
            chdir: "{{ op_agent_temp_dir.path }}"
          register: expected_checksum_agent
          changed_when: false

        - name: Calculate actual checksum of downloaded op-agent binary
          ansible.builtin.shell:
            cmd: sha256sum "{{ op_agent_binary_name }}" | awk '{print $1}'
            chdir: "{{ op_agent_temp_dir.path }}"
          register: actual_checksum_agent
          changed_when: false

        - name: Verify op-agent binary checksum
          ansible.builtin.assert:
            that:
              - expected_checksum_agent.stdout == actual_checksum_agent.stdout
            fail_msg: "Checksum verification failed for op-agent binary"
            success_msg: "Checksum verification passed for op-agent"

        - name: Extract expected checksum for op-agent-client
          ansible.builtin.shell:
            cmd: grep "{{ op_agent_client_binary_name }}" checksums.txt | awk '{print $1}'
            chdir: "{{ op_agent_temp_dir.path }}"
          register: expected_checksum_client
          changed_when: false

        - name: Calculate actual checksum of downloaded op-agent-client binary
          ansible.builtin.shell:
            cmd: sha256sum "{{ op_agent_client_binary_name }}" | awk '{print $1}'
            chdir: "{{ op_agent_temp_dir.path }}"
          register: actual_checksum_client
          changed_when: false

        - name: Verify op-agent-client binary checksum
          ansible.builtin.assert:
            that:
              - expected_checksum_client.stdout == actual_checksum_client.stdout
            fail_msg: "Checksum verification failed for op-agent-client binary"
            success_msg: "Checksum verification passed for op-agent-client"

        - name: Move verified op-agent binary to final location
          ansible.builtin.copy:
            src: "{{ op_agent_temp_dir.path }}/{{ op_agent_binary_name }}"
            dest: "{{ op_agent_install_dir }}/op-agent"
            mode: "0755"
            remote_src: true

        - name: Move verified op-agent-client binary to final location
          ansible.builtin.copy:
            src: "{{ op_agent_temp_dir.path }}/{{ op_agent_client_binary_name }}"
            dest: "{{ op_agent_install_dir }}/op-agent-client"
            mode: "0755"
            remote_src: true

        - name: Verify op-agent installation
          ansible.builtin.command:
            cmd: "{{ op_agent_install_dir }}/op-agent --version"
          register: op_agent_verify
          changed_when: false

        - name: Verify op-agent-client installation
          ansible.builtin.command:
            cmd: "{{ op_agent_install_dir }}/op-agent-client --version"
          register: op_agent_client_verify
          changed_when: false

        - name: Confirm installation versions
          ansible.builtin.assert:
            that:
              - op_agent_verify.rc == 0
              - op_agent_client_verify.rc == 0
              - op_agent_verify.stdout | trim == op_agent_version
              - op_agent_client_verify.stdout | trim == op_agent_version
            fail_msg: "op-agent or op-agent-client installation failed or version mismatch"
            success_msg: "op-agent and op-agent-client v{{ op_agent_version }} successfully installed"

        - name: Clean up temporary directory
          ansible.builtin.file:
            path: "{{ op_agent_temp_dir.path }}"
            state: absent
          when: op_agent_temp_dir.path is defined

    - name: op-agent and op-agent-client are already up to date
      ansible.builtin.debug:
        msg: "op-agent and op-agent-client v{{ op_agent_version }} are already installed"
      when: current_version == op_agent_version
```

Then run:

```bash
ansible-playbook op-agent.yaml
```

</details>

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

### Auto-Start on macOS

To start `op-agent` automatically when you log in to macOS, you can add it as a Login Item:

#### Via System Preferences

- Open System Preferences → Users & Groups → Login Items
- Click the `+` button and navigate to your `op-agent` binary
- Add the binary with arguments: `start --non-interactive`

#### Using `launchd`

Create a launch agent plist file at `~/Library/LaunchAgents/com.kossnocorp.op-agent.plist` (replacing `~/.local/bin/op-agent` with the actual path):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.kossnocorp.op-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>~/.local/bin/op-agent</string>
        <string>start</string>
        <string>--non-interactive</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/op-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/op-agent.err</string>
</dict>
</plist>
```

Then load it:

```sh
launchctl load ~/Library/LaunchAgents/com.kossnocorp.op-agent.plist
```

**Note:** Update `~/.local/bin/op-agent` to the actual path of your `op-agent` binary.

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
