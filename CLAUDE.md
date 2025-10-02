# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Simple CI is a tool for running automated builds and tests with a focus on low-level hardware testing. It does not require containers and is designed to run natively on systems. The project uses a YAML config format inspired by Gitlab CI/CD, Drone, and Ansible.

## Architecture

### Core Concepts

**Units**: The system is composed of units. Each unit can trigger additional units, allowing you to start/sequence operations and create build/test pipelines.

**Trigger Units**: Watch for various conditions and trigger when conditions are met. They are typically used to start other units. Types include:
- System booted triggers (fires on first run after boot)
- Git update triggers (fires when Git updates are detected in local workspace)
- Cron triggers (standard cron format)

**Reboot Cycle Tests**: Can be enabled in config files and are typically triggered by system booted units to count boot cycles.

### Implementation Details

**unit.go**: Defines the `Unit` and `TriggerUnit` interfaces that all units must implement.

**bootdetect.go**: Contains the `BootDetector` type that detects system boot time by reading `/proc/uptime` and tracks whether this is the first run since boot using a state file. Uses a 10-second tolerance when comparing boot times to handle calculation variations.

**systembooted.go**: Implements the `SystemBootedTrigger` which uses `BootDetector` to fire once per boot cycle. State files default to `/var/lib/simpleci/systembooted.state` but can be customized.

**config.go**: Handles YAML configuration parsing and unit instantiation. The config format uses a wrapper pattern to support multiple unit types. The `state_location` field is required in all configuration files.

## Development

### Language and Dependencies

- Go 1.25.1
- Module: `github.com/cbrake/simpleci`

### Code Style

The project uses Prettier with the following settings:
- Tabs (not spaces)
- No semicolons
- Arrow function parentheses: always
- Trailing commas: ES5
- Prose wrap: always

### Building and Running

Build the project:
```bash
go build -o simpleci ./cmd/simpleci
```

Run with a config file:
```bash
./simpleci example-config.yaml
```

Run tests:
```bash
go test -v
```

Run a specific test:
```bash
go test -v -run TestSystemBootedTrigger
```
