# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Run units now support optional `shell` configuration field to specify which
  shell to use (bash, sh, etc.). Defaults to 'sh' when not specified.
- Run units now support optional `use_pty` configuration field to wrap commands
  with `script` for pseudo-TTY support. Useful for tools like bitbake that
  require a TTY environment.
- Added `update` command to automatically update BRun to the latest release
  from GitHub.

### Changed

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
