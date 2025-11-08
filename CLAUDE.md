# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BRun is a tool for running automated builds and tests with a focus on low-level hardware testing. It does not require containers and is designed to run natively on systems. The project uses a YAML config format inspired by Gitlab CI/CD, Drone, and Ansible.

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

**systembooted.go**: Implements the `SystemBootedTrigger` which uses `BootDetector` to fire once per boot cycle. State files default to `/var/lib/brun/systembooted.state` but can be customized.

**config.go**: Handles YAML configuration parsing and unit instantiation. The config format uses a wrapper pattern to support multiple unit types. The `state_location` field is required in all configuration files. Supports automatic decryption of SOPS-encrypted config files at runtime.

**cron.go**: Implements the `CronTrigger` which fires based on standard cron schedules. Uses a 60-second tolerance window to handle orchestrator check intervals and system delays. If a scheduled run is missed by more than the tolerance window (e.g., due to system downtime), that run is skipped to prevent catch-up behavior. Saves the scheduled time (minute boundary) rather than the current time to prevent double-triggering on subsequent checks within the same minute.

**CheckMode Pattern**: The `TriggerUnit` interface uses a `CheckMode` parameter to differentiate between orchestrator polling (`CheckModePolling`) and explicit triggering by another unit (`CheckModeManual`). This allows triggers like GitTrigger to behave differently based on context:
- **Polling mode**: Git units without a `poll` interval skip checks, enabling event-driven workflows
- **Manual mode**: Git units always check when explicitly triggered, regardless of poll interval
- Other triggers (cron, boot, start, file) behave identically in both modes

## Development

### Language and Dependencies

- Go 1.25.1
- Module: `github.com/cbrake/brun`

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
go build -o brun ./cmd/brun
```

Run with a config file:
```bash
./brun example-config.yaml
```

Run tests:
```bash
go test -v
```

Run a specific test:
```bash
go test -v -run TestSystemBootedTrigger
```

## Security

### Secrets Management with SOPS

BRun supports encrypting configuration files with [SOPS (Secrets OPerationS)](https://github.com/getsops/sops). This allows you to store sensitive data like passwords, API keys, and tokens directly in your config files while keeping them encrypted at rest.

**Features:**
- Automatic transparent decryption at runtime
- No changes to user interface - just run `./brun run config.yaml`
- Backward compatible with plaintext configs
- Supports multiple key providers (age, PGP, AWS KMS, GCP KMS, Azure Key Vault)

**Setup:**

1. Install SOPS and age:
```bash
# Install SOPS (see https://github.com/getsops/sops/releases)
# Install age
```

2. Generate an age key:
```bash
age-keygen -o ~/.config/sops/age/keys.txt
# Save the public key (age1...) for encrypting files
```

3. Encrypt your config file:
```bash
sops --encrypt --age <public-key> --in-place config.yaml
```

4. Run BRun normally:
```bash
./brun run config.yaml  # Decrypts automatically if key is available
```

**Key Management:**

SOPS automatically looks for keys in standard locations:
- **age keys:** `~/.config/sops/age/keys.txt`
- **PGP keys:** GPG keyring
- **Cloud KMS:** Uses cloud provider credentials

**Example encrypted config:**

After encrypting, your config file will contain encrypted values with a `sops:` metadata section at the bottom. The file structure remains visible, but sensitive values are encrypted:

```yaml
config:
  state_location: ENC[AES256_GCM,data:...,iv:...,tag:...,type:str]
units:
  - run:
      name: deploy
      script: ENC[AES256_GCM,data:...,iv:...,tag:...,type:str]
sops:
  age:
    - recipient: age1...
      enc: |
        -----BEGIN AGE ENCRYPTED FILE-----
        ...
        -----END AGE ENCRYPTED FILE-----
```

BRun detects the `sops:` metadata and automatically decrypts the file before parsing. Plaintext configs continue to work without modification.
