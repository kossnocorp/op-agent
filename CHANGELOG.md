# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning].

This change log follows the format documented in [Keep a CHANGELOG].

[semantic versioning]: http://semver.org/
[keep a changelog]: http://keepachangelog.com/

## v0.2.2 - 2025-08-21

### Fixed

- Fixed approved commands string arguments losing argument boundaries, i.e., `--format json "age key"` being stored as `--format json age key`.

## v0.2.1 - 2025-08-20

### Fixed

- Fixed `op-agent-client` not handling `--version` and other flags properly.

## v0.2.0 - 2025-08-20

### Added

- Added `op-agent approve` command to pre-approve 1Password CLI commands.

### Changed

- `op-agent --version` (as well as `op-agent-client`) now prints only the version (e.g., `0.2.0`) instead of `op-agent version 0.2.0`.

## v0.1.0 - 2025-08-20

Initial version
