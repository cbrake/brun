# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
