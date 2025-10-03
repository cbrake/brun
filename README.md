# BRun

```
    ╔═══════════════════════════════════╗
    ║                                   ║
    ║   ▄▄▄▄                            ║
    ║   █   █  ██▄▄  █  █  █▄▄          ║
    ║   ████   █  █  █  █  █  █         ║
    ║   █   █  █  █  █  █  █  █         ║
    ║   ████   ██▀   ████  █  █         ║
    ║                                   ║
    ║   ∿∿∿∿ Build → Test → Deploy ∿∿∿∿ ║
    ║  ∿∿∿∿∿∿  Trigger  →  Run  ∿∿∿∿∿∿  ║
    ║                                   ║
    ╚═══════════════════════════════════╝
```

BRun is a tool to run automated builds/tests with a focus on Linux bare-machine
testing. Features/goals:

- simplicity
- composed of chainable units
- emphasis is on automated testing that can have various triggers and
  intelligently log and notifiy
- build in commands for a lot of stuff you might need
- focus on low level testing
- first priority is to run native
- does not require containers (but may support them in the future)
- simple YAML config format
- fast
- built-in commands for common tasks like boot, cron, email, logging

## Example Configuration

Here's a complete example showing all supported unit types:

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  # Start trigger - fires every time brun runs
  - start:
      name: on-start
      on_success:
        - build

  # Boot trigger - fires once per boot cycle
  - boot:
      name: on-boot
      on_success:
        - build
        - test
      always:
        - log-boot

  # Run unit - executes shell commands/scripts
  - run:
      name: build
      directory: /home/user/project
      script: |
        echo "Building project..."
        go build -o brun ./cmd/brun
        echo "Build complete"
      on_success:
        - test
      on_failure:
        - log-build-error

  # Run unit - run tests
  - run:
      name: test
      script: |
        echo "Running tests..."
        go test -v
      on_success:
        - log-success
      on_failure:
        - log-test-error

  # Log unit - write to log files
  - log:
      name: log-boot
      file: /var/log/brun/boot.log

  - log:
      name: log-success
      file: /var/log/brun/success.log

  - log:
      name: log-build-error
      file: /var/log/brun/build-errors.log

  - log:
      name: log-test-error
      file: /var/log/brun/test-errors.log

  # Reboot unit - reboot the system (for reboot cycle testing)
  - reboot:
      name: reboot-system
      delay: 5
```

## Install

To install, download the latest release and then run `brun install`.

If this is run as root, it installs a systemd service that runs as root,
otherwise as the user that runs the install.

If a config file does not exist, one is created.

## Running

Build the project:

```bash
go build -o brun ./cmd/brun
```

Run BRun with a configuration file:

- `brun run config.yaml`: (run the program)
- `brun install`: (install and setup the program)

BRun can be configured for a one-time run (default), or a long running process
that continually looks for triggers.

**One-time run (current implementation):**

The current implementation runs once, checks all trigger conditions, executes
any units whose conditions are met, and then exits. This is suitable for:

- Running from cron
- Manual invocation
- Testing configurations

**Long-running mode (planned):**

In the future, BRun will support a daemon mode that continuously monitors
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

BRun uses a single common state file (YAML format) where all units store state
between runs. This unified approach simplifies state management and makes it
easy to:

- Track all unit state in one location
- Back up and restore state atomically
- Clear all state with a single file deletion
- Inspect and debug state using standard YAML tools

The state file location must be set in the BRun config file.

**State Data:**

Units store different types of state information in the YAML file:

- **Boot trigger**: Last boot time (RFC3339 timestamp)
- **Boot trigger**: boot count
- **Git trigger**: Last processed commit hash
- **Cron trigger**: Last execution time

**State File Format:**

The state file uses YAML format for consistency with the configuration file.
Each unit stores its state under a key corresponding to its name or type.

The state file is automatically created with appropriate permissions (0644) when
BRun runs for the first time.

## File format

YAML is used for BRun file format and leverages the best of Gitlab CI/CD, Drone,
Ansible, and other popular systems.

The system is composed of units. Each unit can trigger additional units. This
allows us to start/sequence operations and create build/test pipelines.

### Config

The BRun file consists of a required `config` section with the following fields:

```yaml
config:
  state_location: /var/lib/brun/state.yaml
```

**Fields:**

- **state_location** (required): Path to the state file where units store their
  state between runs.
  - Defaults to `/var/lib/brun/state.yaml` for root installs
  - Defaults to `~/.config/brun/state.yaml` for user installs

The config file also contains a `units` section as described below.

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

### Start Unit

The Start trigger always fires when brun runs. This can be used to trigger other
units every time the program executes, regardless of boot state or other
conditions.

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - start:
      name: start-trigger
      on_success:
        - build-unit
        - test-unit
```

**Fields:**

- **name** (required): Unique identifier for the start trigger
- **on_success**, **on_failure**, **always** (optional): Standard trigger fields

**Behavior:**

- Always triggers on every brun run
- Does not maintain any state
- Useful for unconditional execution pipelines

### Boot Unit

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
config:
  state_location: /var/lib/brun/state.yaml

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

### Run Unit

The Run unit executes arbitrary shell commands or scripts. This is the primary
execution unit for running builds, tests, or any other commands. The exit code
determines success or failure, which then triggers the appropriate units.

Multiple Run units can be defined in a configuration file to create build and
test pipelines.

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - boot:
      name: boot-trigger
      on_success:
        - build

  - run:
      name: build
      directory: /home/user/project
      script: |
        go build -o brun ./cmd/brun
        go test -v
      on_success:
        - deploy
      on_failure:
        - notify-failure

  - run:
      name: deploy
      script: |
        ./deploy.sh
```

**Fields:**

- **name** (required): Unique identifier for the run unit
- **script** (required): Shell commands to execute. Can be a single command or a
  multi-line script
- **directory** (optional): Working directory where the script will be executed.
  Defaults to the directory where brun was invoked
- **on_success**, **on_failure**, **always** (optional): Standard trigger fields

**Behavior:**

- The script is executed using the system shell
- Exit code 0 is considered success and triggers `on_success` units
- Non-zero exit codes are considered failures and trigger `on_failure` units
- Both stdout and stderr are logged

### Log Unit

The Log unit writes log entries to a file. This is useful for recording events,
errors, or other information during pipeline execution. The log file is created
if it doesn't exist, and entries are appended with timestamps.

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - start:
      name: start-trigger
      on_success:
        - build
      always:
        - log-run

  - run:
      name: build
      script: |
        go build -o brun ./cmd/brun
      on_failure:
        - log-error

  - log:
      name: log-run
      file: /var/log/brun/pipeline.log

  - log:
      name: log-error
      file: /var/log/brun/errors.log
```

**Fields:**

- **name** (required): Unique identifier for the log unit
- **file** (required): Path to the log file where entries will be written
- **on_success**, **on_failure**, **always** (optional): Standard trigger fields

**Behavior:**

- Creates the log file and parent directories if they don't exist
- Appends log entries with timestamps
- File permissions are set to 0644
- Directory permissions are set to 0755

### Git Unit (todo)

A Git trigger is generated when a Git update is detected in a local workspace.

### Cron Unit (todo)

A Cron trigger unit is configured using the standard Unit cron format.

### Email Unit (todo)

Can be used to email the results of a unit.

### Reboot Unit

The reboot unit logs and reboots the system. This is typically used in reboot
cycle testing where the boot trigger can count boot cycles and trigger test
sequences.

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - reboot:
      name: reboot-system
      delay: 5 # optional delay in seconds before reboot (default: 0)
```

**Fields:**

- **name** (required): Unique identifier for the reboot unit
- **delay** (optional): Number of seconds to wait before executing reboot
  (default: 0 for immediate reboot)
- **on_success**, **on_failure**, **always** (optional): Standard trigger fields
  (though typically not used since reboot terminates execution)
