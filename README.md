# Simple CI

Simple CI is a tool to run automated builds/tests. Features/goals:

- focus is low level hardware testing
- does not require containers (but may support them in the future)
- simple YAML config format
- designed first to run native
- can run tests/builds on a local workstation

## Running

Build the project:

```bash
go build -o simpleci ./cmd/simpleci
```

Run Simple CI with a configuration file:

```bash
./simpleci config.yaml
```

Simple CI can be configured for a one-time run (default), or a long running
process that continually looks for triggers.

**One-time run (current implementation):**

The current implementation runs once, checks all trigger conditions, executes
any units whose conditions are met, and then exits. This is suitable for:

- Running from cron
- Manual invocation
- Testing configurations

**Long-running mode (planned):**

In the future, Simple CI will support a daemon mode that continuously monitors
trigger conditions and executes units when triggered. This will be suitable for:

- System service deployment
- Continuous monitoring of git repositories
- Scheduled cron-based execution

## Logging

By default, logging is sent to STDOUT, and each unit logs:

- logs when it triggers or runs
- any errors

Additional log units can log specific events.

## State

Simple CI uses a single common state file (YAML format) where all units store
state between runs. This unified approach simplifies state management and makes
it easy to:

- Track all unit state in one location
- Back up and restore state atomically
- Clear all state with a single file deletion
- Inspect and debug state using standard YAML tools

**State File Location:**

By default, the state file is located at `/var/lib/simpleci/state.yaml`. This
location can be configured via command-line arguments or environment variables.

**State Data:**

Units store different types of state information in the YAML file:

- **Boot trigger**: Last boot time (RFC3339 timestamp)
- **Git trigger**: Last processed commit hash
- **Cron trigger**: Last execution time
- **Reboot cycle test**: Test iteration count and status

**State File Format:**

The state file uses YAML format for consistency with the configuration file.
Each unit stores its state under a key corresponding to its name or type.

The state file is automatically created with appropriate permissions (0644) when
Simple CI runs for the first time.

## File format

YAML is used for Simple CI file format and leverages the best of Gitlab CI/CD,
Drone, Ansible, and other popular systems.

The system is composed of units. Each unit can trigger additional units. This
allows us to start/sequence operations and create build/test pipelines.

### Config

The Simple CI file consists of an optional config section with the following
fields:

- state_location (optional, defaults to `/var/lib/simpleci/state.yaml`)

## Units

### Common Unit Fields

All units share the following common fields:

- **name** (required): A unique identifier for the unit. This name is used to
  reference the unit when triggering it from other units.
- **on_success** (optional): An array of unit names to trigger when this unit
  completes successfully.
- **on_failure** (optional): An array of unit names to trigger when this unit
  fails.
- **always** (optional): An array of unit names to trigger regardless of whether
  this unit succeeds or fails. These units run after success/failure triggers.

### Boot

The boot unit triggers if this is the first time the program has been run since
the system booted. The boot unit stores the last boot time in the common state
file.

**How it works:**

The boot trigger detects boot events by:

1. Reading `/proc/uptime` to calculate the system boot time
2. Comparing this with a stored boot time from the previous run (saved in the
   common state file)
3. Triggering when the boot times differ by more than 10 seconds

**Configuration example:**

```yaml
units:
  - boot:
      name: boot-trigger
      on_success:
        - build-unit
        - test-unit
```

When the boot trigger fires successfully, it will trigger the units listed in
`on_success` (in this example, `build-unit` and `test-unit`).

The boot time is automatically stored in the common state file under the unit's
name.

### Git

A Git trigger is generated when a Git update is detected in a local workspace.

### Cron

A Cron trigger unit is configured using the standard Unit cron format.

## Reboot

The reboot unit logs and reboots the unit.
