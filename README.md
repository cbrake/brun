# BRun

```
 ____  ____
| __ )|  _ \ _   _ _ __
|  _ \| |_) | | | | '_ \
| |_) |  _ <| |_| | | | |
|____/|_| \_\\__,_|_| |_|

~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~
  Build → Test → Deploy
~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~ ~
```

BRun is a tool to run automated builds/tests with a focus on Linux bare-machine
testing. Features/goals:

- simplicity
- composed of chainable units
- emphasis is on automated testing that can have various triggers and
  intelligently log and notify
- built-in commands for a lot of stuff you might need
- focus on low level testing
- first priority is to run native
- does not require containers (but may support them in the future)
- simple YAML config format
- fast
- built-in commands for common tasks like boot, cron, email, logging

## Example Configuration

Here's a complete example showing all supported unit types:

```yaml
konfig:
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

  # Count unit - track how many times units trigger
  - count:
      name: build-counter

  # Cron trigger - runs every 5 minutes (useful in daemon mode)
  - cron:
      name: periodic-check
      schedule: "*/5 * * * *"
      on_success:
        - test

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

## Usage

```
Usage: ./brun <command> [args]

Commands:
  run <config-file> [-daemon]    Run brun with the given config file
                                  -daemon: run in daemon mode (continuous monitoring)
                                  -unit <unit name>: run a single unit (useful for debugging)
                                   Triggers are not executed.
                                  -trigger <unit name>: trigger the named unit and execute
                                   triggers.
  install                        Install brun as a systemd service
```

**One-time run:**

By default, BRun runs once, checks all trigger conditions, executes any units
whose conditions are met, and then exits. This is suitable for:

- Running from external cron
- Manual invocation
- Testing configurations

```bash
brun run config.yaml
```

**Daemon mode:**

BRun supports a daemon mode that continuously monitors trigger conditions and
executes units when triggered. In this mode, triggers are checked every 10
seconds. This is suitable for:

- System service deployment
- Continuous monitoring with cron triggers
- Long-running background processes

```bash
brun run config.yaml -daemon
```

## Circular Dependency Protection

BRun protects against circular dependencies when units trigger each other. For
example, if Unit A triggers Unit B, and Unit B triggers Unit A, this could cause
an infinite loop.

**How it works:**

- The orchestrator maintains a results map that tracks which units have executed
  in the current trigger cycle
- Before executing a unit, the orchestrator checks if it has already run in this
  cycle
- If a unit has already executed, it is skipped to prevent circular dependencies
- At the start of each trigger cycle (every 10 seconds in daemon mode), the
  results map is cleared, allowing units to run again in the next cycle

This approach allows:

- **Periodic triggers to work correctly**: Units can be triggered multiple times
  across different cycles (e.g., cron triggers firing every minute)
- **Circular dependency protection**: Within a single trigger cycle, units
  cannot trigger each other recursively

**Example:**

```yaml
units:
  - cron:
      name: every-minute
      schedule: "* * * * *"
      on_success:
        - task-a

  - run:
      name: task-a
      script: echo "Task A"
      always:
        - task-b

  - run:
      name: task-b
      script: echo "Task B"
      always:
        - task-a # This would create a circular dependency
```

In this example:

- The cron trigger fires every minute and triggers `task-a`
- `task-a` triggers `task-b`
- `task-b` attempts to trigger `task-a`, but it's already in the results map
- The circular trigger is prevented, and the log shows: "Unit 'task-a' already
  executed in this chain, skipping to prevent circular dependency"
- In the next minute, the results map is cleared and the cycle can run again

## Logging

By default, logging is sent to STDOUT, and each unit logs:

- when it triggers or runs
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

- **Boot trigger**: Last boot time (RFC3339 timestamp) and boot count
- **Cron trigger**: Last execution time (RFC3339 timestamp)
- **Count unit**: Trigger counts per triggering unit
- **Git trigger**: Last processed commit hash (todo)

**State File Format:**

The state file uses YAML format for consistency with the configuration file.
Each unit stores its state under a key corresponding to its name or type.

The state file is automatically created with appropriate permissions (0644) when
BRun runs for the first time.

## File format

YAML is used for the BRun file format and leverages the best of Gitlab CI/CD,
Drone, Ansible, and other popular systems.

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

**Behavior:**

- Always triggers on every brun run
- Does not maintain any state
- Useful for unconditional execution pipelines

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

### Boot Unit

The boot unit triggers if this is the first time the program has been run since
the system booted. The boot unit stores the last boot time in the common state
file.

**Behavior:**

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

**Fields:**

- **script** (required): Shell commands to execute. Can be a single command or a
  multi-line script
- **directory** (optional): Working directory where the script will be executed.
  Defaults to the directory where brun was invoked
- **timeout** (optional): timeout duration for the task to complete (e.g.,
  "30s", "5m", "1h", "1h30m"). If no timeout is specified, it runs until
  completion. If the task times out, an error message is logged.

**Behavior:**

- The script is executed using the system shell
- Exit code 0 is considered success and triggers `on_success` units
- Non-zero exit codes are considered failures and trigger `on_failure` units
- Both stdout and stderr are logged

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

### Log Unit

The Log unit writes log entries to a file. This is useful for recording events,
errors, or other information during pipeline execution. The log file is created
if it doesn't exist, and entries are appended with timestamps.

**Fields:**

- **file** (required): Path to the log file where entries will be written

**Behavior:**

- Creates the log file and parent directories if they don't exist
- Appends log entries with timestamps
- File permissions are set to 0644
- Directory permissions are set to 0755

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

### Count Unit

The Count unit creates an entry in the state file for every unit that triggers
this unit and counts how many times it has been triggered. This is useful for
tracking how often specific events occur or how many times particular units
execute.

**Behavior:**

- Tracks separate counts for each unit that triggers it
- Stores counts in the state file under the count unit's name
- Each triggering unit has its own counter
- Counts persist across runs

**State File Format:**

The count unit stores data in the state file like this:

```yaml
count-runs:
  start-trigger: 5

count-builds:
  build: 3

count-failures:
  build: 1
```

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
        - count-runs

  - run:
      name: build
      script: |
        go build -o brun ./cmd/brun
      on_success:
        - count-builds
      on_failure:
        - count-failures

  - count:
      name: count-runs

  - count:
      name: count-builds

  - count:
      name: count-failures
```

### Cron Unit

The Cron unit is a trigger that fires based on a cron schedule. It uses the
standard cron format to define when the trigger should activate. In daemon mode,
the trigger is checked every 10 seconds. The
[robfig/cron](https://pkg.go.dev/github.com/robfig/cron/v3) package is used for
schedule parsing.

**Fields:**

- **schedule** (required): Cron schedule in standard format (minute hour day
  month weekday)

**Behavior:**

- Triggers based on the cron schedule
- Stores last execution time in the state file
- Works in both one-time and daemon modes
- In one-time mode: triggers if schedule indicates it should have run since last
  execution
- In daemon mode: continuously monitors and triggers at scheduled times

**Cron Schedule Format:**

Standard 5-field cron format:

```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-6, Sunday=0)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

Examples:

- `* * * * *` - Every minute
- `*/5 * * * *` - Every 5 minutes
- `0 2 * * *` - Daily at 2:00 AM
- `30 14 * * 1-5` - Weekdays at 2:30 PM
- `0 0 1 * *` - First day of every month at midnight

**State File Format:**

The cron unit stores the last execution time:

```yaml
daily-backup:
  last_execution: "2025-10-03T02:30:00-04:00"

health-check:
  last_execution: "2025-10-03T18:00:00-04:00"
```

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  # Cron trigger - runs every day at 2:30 AM
  - cron:
      name: daily-backup
      schedule: "30 2 * * *"
      on_success:
        - backup-unit

  # Cron trigger - runs every 5 minutes
  - cron:
      name: health-check
      schedule: "*/5 * * * *"
      on_success:
        - check-services

  - run:
      name: backup-unit
      script: |
        echo "Running daily backup..."
        # backup commands here

  - run:
      name: check-services
      script: |
        echo "Checking services..."
        # health check commands here
```

### Git Unit (todo)

A Git trigger is generated when a Git update is detected in a local workspace.

### Email Unit (todo)

Can be used to email the results of a unit.

### Reboot Unit

The reboot unit logs and reboots the system. This is typically used in reboot
cycle testing where the boot trigger can count boot cycles and trigger test
sequences.

**Fields:**

- **delay** (optional): Number of seconds to wait before executing reboot
  (default: 0 for immediate reboot)

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - reboot:
      name: reboot-system
      delay: 5 # optional delay in seconds before reboot (default: 0)
```
