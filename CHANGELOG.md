# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.12] - 2025-10-14

### Fixed

- Trigger units (git, cron, file, etc.) now properly check their conditions when
  triggered by other units via `on_success`, `on_failure`, or `always` fields.
  For example, when a cron unit triggers a git unit, the git unit will only
  execute if there are actual git updates, preventing unnecessary builds and
  operations.

## [0.0.11] - 2025-10-13

- test release

## [0.0.10] - 2025-10-13

- Email notifications now include a proper Date header, ensuring emails display
  correct timestamps in mail clients and comply with email standards (RFC 5322).

## [0.0.9] - 2025-10-08

### Added

- Graceful shutdown now waits for active units to complete when you press Ctrl+C
  or send a termination signal. BRun will wait for running units to finish
  before exiting, preventing interruption of critical operations like builds,
  deploys, or backups.

## [0.0.8] - 2025-10-08

### Fixed

- State file can now be specified without a directory path (e.g., `state.yaml`),
  eliminating crashes when using simple filenames.
- Boot counters and other state-dependent units now work more reliably by
  loading state once at startup instead of repeatedly during execution,
  preventing race conditions and improving performance.
- Trigger units now execute more efficiently by avoiding redundant condition
  checks, improving overall system responsiveness.
- Multiple units can now trigger the same unit (such as email or log units)
  multiple times in a single execution chain, while still properly detecting and
  preventing circular dependencies. This allows for more flexible notification
  and logging patterns.

## [0.0.7] - 2025-10-07

### Added

- Version information now displays when running brun, helping you track which
  version you're using.
- Git trigger now supports `poll` field to set a custom polling interval (e.g.,
  "2m", "30s", "5m"), giving you control over how often repositories are checked
  for updates.
- Git trigger now supports `debug` field to enable detailed logging of git
  operations (fetch, reset, submodule updates) when troubleshooting.

### Changed

- Git trigger polling messages are now hidden by default, reducing log noise
  during continuous monitoring.

## [0.0.6] - 2025-10-07

### Added

- Git trigger now automatically updates local workspaces when monitoring
  repositories, keeping your workspace in sync with the remote.
- Git trigger supports `branch` field to specify which branch to monitor and
  update.
- Git trigger supports optional `reset` field to force reset workspace to remote
  state, discarding local changes.
- Git trigger automatically updates submodules recursively when updating
  workspaces.
- Email units now support `limit_lines` field to limit email output to the last
  N lines, preventing overwhelming emails from verbose build processes.

### Fixed

- Output captured from run units now strips ANSI escape sequences (colors,
  cursor movements) from emails and logs while preserving them in terminal
  display, making automated build logs much more readable.

## [0.0.5] - 2025-10-07

### Added

- Run units now support optional `shell` configuration field to specify which
  shell to use (bash, sh, etc.). Defaults to 'sh' when not specified.
- Run units now support optional `use_pty` configuration field to wrap commands
  with `script` for pseudo-TTY support. Useful for tools like bitbake that
  require a TTY environment.
- Added `update` command to automatically update BRun to the latest release from
  GitHub.

### Fixed

- Fixed PTY command execution to properly handle multiline scripts and prevent
  newline characters from being interpreted literally.

## [0.0.4] - 2025-10-06

- Add version command.

## [0.0.3] - 2025-10-06

- Releases are now a single binary instead of archive. This makes it easier to
  install.

## [0.0.2] - 2025-10-06

### Added

- Install command now supports `-daemon` flag to install systemd service in
  daemon mode
- Improved help output with better formatting, alignment, and examples section

### Changed

- Updated systemd service files to use `Type=simple` and `Restart=always` when
  installed with `-daemon` flag
- Improved CLI help text to match industry-standard tools with clearer structure

## [0.0.1] - 2025-10-06

- initial release
