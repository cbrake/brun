# BRun

<p align="center">
  <img src="assets/brun-logo.png" alt="BRun Logo" width="400">
</p>

<p align="center">
<em>Trigger → Run</em>
</p>

Do you find tools like GitHub Actions or Ansible useful, but would like a simple
way to do similar things natively? BRun is a native Linux automation tool that
connects triggers (boot, cron, file changes, git commits) to actions (run
scripts, send emails, log events, reboot). Build CI/CD pipelines, automate
system tasks, or test embedded devices—all with a single binary and no
dependencies.

**Features/goals:**

- ✨ **simple!!!**
- ⚡ **fast!!!**
- 📦 no dependencies -- [install](#example-install-on-linux) a single statically
  linked binary and go for it ...
- 🛠️ built-in [units](#units) for common tasks like boot, scripts, cron, email,
  git, file watching
- 🔗 units can be chained into pipelines
- 💻 first priority is to run native
- 🚫 does not require containers (but may support them in the future)
- 📄 simple YAML [config format](#file-format)
- 🔒 built-in [secrets management](#secrets-management) with SOPS encryption

**Things you might do with BRun**

- Reboot cycle test for embedded systems.
- Nightly Yocto builds on your powerful workstation.
- Run admin tasks like backups.
- Monitor the `/etc` directory a server for changes.
- Implemented a watchdog that reboots the system under certain conditions.
- Run build/test/deploy pipelines.
- Notify someone when CPU usage is too high or disk space too low.

## Table of Contents

<!--toc:start-->

- [BRun](#brun)
  - [Example Configuration](#example-configuration)
  - [Install](#install)
    - [Example install on Linux:](#example-install-on-linux)
    - [Autostart with systemd](#autostart-with-systemd)
    - [Updating](#updating)
  - [Usage](#usage)
  - [Circular Dependency Protection](#circular-dependency-protection)
  - [Logging](#logging)
  - [State](#state)
  - [Secrets Management](#secrets-management)
  - [File format](#file-format)
    - [Config](#config)
  - [Units](#units)
    - [Common Unit Fields](#common-unit-fields)
    - [Start Unit](#start-unit)
    - [Boot Unit](#boot-unit)
    - [Run Unit](#run-unit)
    - [Log Unit](#log-unit)
    - [Count Unit](#count-unit)
    - [Cron Unit](#cron-unit)
    - [File Unit](#file-unit)
    - [Git Unit](#git-unit)
    - [Email Unit](#email-unit)
    - [Email Receive Unit (TODO)](#email-receive-unit-todo)
    - [Reboot Unit](#reboot-unit)
  - [Program lifecycle](#program-lifecycle)
  - [Status](#status)
  <!--toc:end-->

## Example Configuration

Here's an example showing how various units are specified and interact (see also
more [examples](examples) and our own [dogfood](build.yaml)):

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

  # Count unit - track how many times units trigger
  - count:
      name: build-counter

  # Cron trigger - runs every 5 minutes (useful in daemon mode)
  - cron:
      name: periodic-check
      schedule: "*/5 * * * *"
      on_success:
        - test

  # File trigger - monitors source files for changes (daemon mode)
  - file:
      name: watch-files
      pattern: "**/*.go"
      on_success:
        - build
        - test

  # Git trigger - monitors repository for changes
  - git:
      name: watch-repo
      repository: /home/user/project
      branch: main
      poll: 2m
      on_success:
        - build

  # Email unit - send notifications
  - email:
      name: email-failure
      to:
        - admin@example.com
      from: brun@example.com
      subject: "Build/Test Failure"
      smtp_host: smtp.gmail.com
      smtp_port: 587
      smtp_user: brun@example.com
      smtp_password: your-app-password
      smtp_use_tls: true
      include_output: true

  # Reboot unit - reboot the system (for reboot cycle testing)
  - reboot:
      name: reboot-system
      delay: 5
```

## Install

To install, download the
[latest release](https://github.com/cbrake/brun/releases) binary.

### Example install on Linux:

Copy and paste the following into your terminal:

```
export VER=0.0.9
export ARCH=$(uname -m)
# Convert aarch64 to arm64 to match release binary names
[ "$ARCH" = "aarch64" ] && ARCH="arm64"
export BINARY=brun-v${VER}-Linux-${ARCH}
wget https://github.com/cbrake/brun/releases/download/v${VER}/${BINARY}
# Install to ~/.local/bin for user, /usr/local/bin for root
if [ "$(id -u)" -eq 0 ]; then
  mkdir -p /usr/local/bin
  install -m 755 ${BINARY} /usr/local/bin/brun
else
  mkdir -p ~/.local/bin
  install -m 755 ${BINARY} ~/.local/bin/brun
fi
rm ${BINARY}
```

### Autostart with systemd

If you would like to install a systemd unit to run brun automatically, then run:

`brun install` (run brun once then exit)

or

`brun install -daemon` (run in daemon mode)

If `brun install` is run as root, it installs a systemd service that runs as
root, otherwise as the user who runs the install.

If a config file does not exist, one is created.

### Updating

After initial installation, the `brun update` command can be used to update to
the latest release.

## Usage

```
Usage: brun COMMAND [OPTIONS]

Commands:
  run <config-file>       Run brun with the given config file
  install                 Install brun as a systemd service
  update                  Updates BRun to the latest version
  version                 Display version information

Run Options:
  -daemon                 Run in daemon mode (continuous monitoring)
  -unit <name>            Run a single unit (triggers disabled, useful for debugging)
  -trigger <name>         Trigger a unit and execute its on_success triggers

Install Options:
  -daemon                 Install service in daemon mode (continuous monitoring)

Examples:
  brun run config.yaml
  brun run config.yaml -daemon
  brun run config.yaml -unit my-build
  brun install
  brun install -daemon
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

- The orchestrator tracks the current execution path (call stack) as units
  trigger each other
- Before executing a unit, the orchestrator checks if it's already in the
  current call stack
- If a unit is already in the call stack, it is skipped to prevent circular
  dependencies
- Units can be triggered multiple times in the same execution as long as they're
  not in a circular loop

This approach allows:

- **Flexible trigger chains**: The same unit (like an email or log unit) can be
  triggered multiple times from different branches in a single execution
- **Circular dependency protection**: Units cannot trigger themselves directly
  or indirectly through other units in the same execution path

**Example - Circular dependency prevented:**

```yaml
units:
  - start:
      name: start-trigger
      on_success:
        - task-a

  - run:
      name: task-a
      script: echo "Task A"
      on_success:
        - task-b

  - run:
      name: task-b
      script: echo "Task B"
      on_success:
        - task-a # This would create a circular dependency
```

In this example:

- `start-trigger` triggers `task-a`
- `task-a` triggers `task-b`
- `task-b` attempts to trigger `task-a`, but it's already in the call stack
- The circular trigger is prevented, and the log shows: "Unit 'task-a' already
  in call stack, skipping to prevent circular dependency"

**Example - Multiple triggers allowed:**

```yaml
units:
  - start:
      name: start-trigger
      on_success:
        - build-frontend
        - build-backend

  - run:
      name: build-frontend
      script: npm run build
      always:
        - notify-team

  - run:
      name: build-backend
      script: go build ./...
      always:
        - notify-team

  - email:
      name: notify-team
      to:
        - team@example.com
      # ... email config ...
```

In this example:

- Both `build-frontend` and `build-backend` trigger `notify-team`
- The `notify-team` email unit runs twice (once from each build)
- This is allowed because `notify-team` is not in a circular dependency

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
- **File trigger**: File hashes for change detection
- **Git trigger**: Last processed commit hash

**State File Format:**

The state file uses YAML format for consistency with the configuration file.
Each unit stores its state under a key corresponding to its name or type.

The state file is automatically created with appropriate permissions (0644) when
BRun runs for the first time.

## Secrets Management

BRun supports encrypting configuration files with
[SOPS (Secrets OPerationS)](https://github.com/getsops/sops), allowing you to
safely store passwords, API keys, and other sensitive data directly in your
config files.

**Benefits:**

- Keep secrets encrypted at rest in version control
- Transparent decryption at runtime - no UI changes needed
- Support for multiple key providers (age, PGP, AWS KMS, GCP KMS, Azure Key
  Vault)
- Backward compatible - plaintext configs still work

**Quick Start:**

1. Install [SOPS](https://github.com/getsops/sops/releases) and
   [age](https://github.com/FiloSottile/age/releases)

2. Generate an encryption key:

```bash
age-keygen -o ~/.config/sops/age/keys.txt
# Save the public key (age1...) shown in output
```

3. Encrypt your config file:

```bash
sops --encrypt --age <your-public-key> --in-place config.yaml
```

4. Run BRun normally:

```bash
brun run config.yaml  # Automatically decrypts
```

Your secrets are now encrypted in the config file but decrypted transparently
when BRun runs. The file structure remains visible (unit names, triggers, etc.),
only sensitive values are encrypted.

**Selective Field Encryption:**

You can configure SOPS to encrypt only sensitive fields (like passwords and API
keys) while keeping the rest of your config readable. Create a `.sops.yaml` file
in your repository root:

```yaml
creation_rules:
  - path_regex: \.yaml$
    encrypted_regex: "^(.*password.*|.*secret.*|.*key.*|.*token.*|smtp_user)$"
    age: your-public-key-here
```

This will encrypt only fields matching the patterns (password, secret, key,
token, etc.) while leaving structural fields like `name`, `script`, and
`directory` in plaintext for easy review in version control.

See [.sops.yaml](.sops.yaml) for a complete example configuration.

## File format

YAML is used for the BRun config file and is similar to config files used in
Gitlab CI/CD, Drone, Ansible, etc.

The configuration is composed of chainable units. Each unit can trigger
additional units. This allows us to start/sequence operations and create
workflow pipelines.

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

**Variables**

_(NOTE: Variables are in planning phase and have not been implemented yet.)_

Variables can be defined in a `vars` block and referenced in any string field
using [Go Templates](https://pkg.go.dev/text/template). Variables are expanded
when a unit is run so that variable updates are processed.

**Syntax:**

Variables are accessed using Go template syntax with double curly braces:

```yaml
vars:
  project_name: myapp
  build_dir: /home/user/builds
  version: 1.0.0

units:
  - run:
      name: build
      directory: { { .build_dir } }
      script: |
        echo "Building {{ .project_name }} version {{ .version }}"
        go build -o {{ .project_name }}
```

**Features:**

- **Basic variables**: Access with `{{ .variable_name }}`
- **Nested variables**: Use dot notation `{{ .config.path }}`
- **Pipelines**: Chain operations `{{ .name | upper | quote }}`
- **Conditionals**:
  `{{ if eq .env "prod" }}production{{ else }}development{{ end }}`
- **Loops**: `{{ range .items }}{{ . }}{{ end }}`
- **Functions**: Built-in functions like `eq`, `ne`, `lt`, `gt`, `and`, `or`,
  `not`

**Example with nested variables:**

Variables can be nested using maps to organize related configuration:

```yaml
vars:
  database:
    host: localhost
    port: 5432
    name: myapp_db
  server:
    host: 0.0.0.0
    port: 8080

units:
  - run:
      name: start-server
      script: |
        echo "Starting server on {{ .server.host }}:{{ .server.port }}"
        echo "Connecting to database {{ .database.name }} at {{ .database.host }}:{{ .database.port }}"
        ./start-server
```

**Example with conditionals:**

```yaml
vars:
  environment: production
  enable_tests: true

units:
  - run:
      name: deploy
      script: |
        echo "Deploying to {{ .environment }}"
        {{ if eq .environment "production" }}
        ./deploy-prod.sh
        {{ else }}
        ./deploy-dev.sh
        {{ end }}

  - run:
      name: test
      {{ if .enable_tests }}
      script: go test -v ./...
      {{ else }}
      script: echo "Tests disabled"
      {{ end }}
```

**Environment variables:**

Environment variables can be accessed using the `env` function (if available):

```yaml
units:
  - run:
      name: build
      script: |
        echo "User: {{ env "USER" }}"
        echo "Home: {{ env "HOME" }}"
```

**Whitespace control:**

Use `-` to trim whitespace before or after template actions:

```yaml
script: |
  {{- if .debug }}
  echo "Debug mode enabled"
  {{- end }}
```

See the [Go template documentation](https://pkg.go.dev/text/template) for full
syntax reference.

## Units

BRun supports the following unit types:

- [Boot Unit](#boot-unit) - Triggers once per boot cycle
- [Count Unit](#count-unit) - Tracks trigger counts
- [Cron Unit](#cron-unit) - Triggers based on cron schedule
- [Email Unit](#email-unit) - Sends email notifications
- [File Unit](#file-unit) - Monitors files for changes
- [Git Unit](#git-unit) - Monitors Git repository for commits
- [Log Unit](#log-unit) - Writes log entries to files
- [Reboot Unit](#reboot-unit) - Reboots the system
- [Run Unit](#run-unit) - Executes shell commands/scripts
- [Start Unit](#start-unit) - Triggers on every program start

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

**Trigger unit behavior:**

When a trigger unit (boot, cron, file, git, start) is triggered by another unit
via `on_success`, `on_failure`, or `always`, the trigger unit's condition is
still checked before execution. For example, if a cron unit triggers a git unit,
the git unit will only execute if there are actual git updates detected. This
prevents unnecessary operations and ensures triggers only fire when their
conditions are truly met.

### Start Unit

The Start trigger always fires when BRun runs. This can be used to trigger other
units every time the program executes, regardless of boot state or other
conditions.

**Behavior:**

- Always triggers on every BRun
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
- **shell** (optional): specify shell to use when running command (bash, etc).
  By default, 'sh' is used.
- **use_pty** (optional): when set to true, wraps the command with `script` to
  provide a pseudo-TTY. This is useful for tools like bitbake that require a TTY
  environment. Default is false.

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

  - run:
      name: bitbake-build
      shell: bash
      use_pty: true
      script: |
        source oe-init-build-env
        bitbake core-image-minimal
      timeout: 2h
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
tracking how often specific events (like errors) occur or how many times
particular units execute. The count quickly tells you something happened, and
then the log files can be examined to understand why.

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

### File Unit

The File unit monitors files and triggers when they are changed. Files can be
specified using glob patterns with support for `**` recursive matching. New or
removed files are detected as changes.

**Fields:**

- **pattern** (required): Glob pattern to match files (supports `**` for
  recursive matching)

**Behavior:**

- Monitors files matching the glob pattern
- Triggers when file content changes (detected via SHA256 hash)
- Triggers when files are added or removed
- Stores file hashes in the state file
- Triggers on first run (initial file state)
- Ignores directories (only monitors regular files)
- Works in both one-time and daemon modes

**Pattern Syntax:**

The file unit supports advanced glob patterns including:

- `*` - matches any sequence of non-separator characters
- `?` - matches any single non-separator character
- `[abc]` - matches any character in the set
- `[a-z]` - matches any character in the range
- `**` - matches zero or more directories recursively

**Pattern Examples:**

- `**/*.go` - all Go files recursively
- `src/**/*.ts` - all TypeScript files under `src/`
- `config/*.yaml` - config files non-recursively
- `**/*.{html,css,js}` - multiple filetypes

**State File Format:**

The file unit stores a hash of all monitored files:

```yaml
watch-source:
  files_state: "file1.go:a1b2c3...|file2.go:d4e5f6..."
```

**Configuration example:**

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  # File trigger - monitors Go source files
  - file:
      name: watch-source
      pattern: "**/*.go"
      on_success:
        - build
        - test

  - run:
      name: build
      script: |
        echo "Building..."
        go build -o app ./cmd/app

  - run:
      name: test
      script: |
        echo "Running tests..."
        go test -v ./...
```

**Daemon mode example:**

When running in daemon mode, the file trigger continuously monitors files and
automatically triggers builds/tests when changes are detected:

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - file:
      name: auto-build
      pattern: "**/*.go"
      on_success:
        - build
        - test
      always:
        - email-notify

  - run:
      name: build
      script: |
        go build -o app ./cmd/app

  - run:
      name: test
      script: |
        go test -v ./...

  - email:
      name: email-notify
      to:
        - team@example.com
      from: ci@example.com
      subject_prefix: "Build Status"
      smtp_host: smtp.example.com
      smtp_port: 587
      smtp_user: ci@example.com
      smtp_password: secret
```

Run with: `brun run config.yaml -daemon`

This creates a continuous integration system that automatically builds and tests
your code whenever source files are modified.

### Git Unit

The Git unit is a trigger that fires when changes are detected in a Git
repository. It monitors the repository's HEAD commit and triggers when new
commits are detected. This is useful for automatically running builds, tests, or
deployments when code changes.

If the `repository` field points to a local Git workspace (vs a Repo URL), the
workspace and submodules are updated to the latest on the specified branch.

**Fields:**

- **repository** (required): Path to the Git repository to monitor
- **branch** (required): Branch to monitor
- **reset** (optional): optionally reset the workspace to the state of the repo
  HEAD (`git reset --hard`)
- **poll** (optional): polling interval for checking repository updates (e.g.,
  "2m", "30s", "1h"). When set, the git unit actively checks for updates at the
  specified interval in daemon mode. When omitted or set to empty string, the
  unit operates in manual trigger mode and only checks for updates when
  triggered by another unit (e.g., via `on_success`).
- **debug** (optional): when true, logs detailed git operation messages (fetch,
  reset, submodule updates). Defaults to false.

**Behavior:**

- Monitors the HEAD commit hash of the specified Git repository
- Triggers when the commit hash changes (new commits detected)
- Stores the last seen commit hash in the state file
- Triggers on first run (initial repository state)
- Uses go-git library (no git CLI tool required)
- Works in both one-time and daemon modes

**State File Format:**

The git unit stores the last seen commit hash:

```yaml
watch-repo:
  last_commit_hash: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
```

**Configuration example:**

When running in daemon mode, the git trigger continuously monitors the
repository and automatically triggers builds/tests when new commits are
detected:

```yaml
config:
  state_location: /var/lib/brun/state.yaml

units:
  - git:
      name: auto-build
      repository: /home/user/project
      branch: main
      poll: 2m # Check for updates every 2 minutes
      debug: false # Suppress verbose git operation logs
      on_success:
        - build

  - run:
      name: build
      directory: /home/user/project
      script: |
        go build -o app ./cmd/app
        go test -v ./...
      always:
        - email

  - email:
      name: email
      to:
        - team@example.com
      from: ci@example.com
      subject_prefix: "Build Success"
      smtp_host: smtp.example.com
      smtp_port: 587
      smtp_user: ci@example.com
      smtp_password: secret
```

This creates a continuous integration system that automatically builds and tests
your code whenever changes are pushed to the repository.

### Email Unit

The Email unit sends email notifications with optional output from triggering
units. This is useful for alerting on build failures, test results, or other
important events. Supports both plain SMTP and STARTTLS encryption.

**Fields:**

- **to** (required): Array of email addresses to send to
- **from** (required): Sender email address
- **subject_prefix** (optional): Email subject line prefix. ':
  <unit-name>:<success|fail>' is appended after prefix and is always included.
- **smtp_host** (required): SMTP server hostname
- **smtp_port** (optional): SMTP server port. Defaults to 587 (submission port)
- **smtp_user** (optional): SMTP username for authentication
- **smtp_password** (optional): SMTP password for authentication
- **smtp_use_tls** (optional): Enable STARTTLS encryption. Defaults to true
- **include_output** (optional): Include captured output from triggering unit.
  Defaults to true
- **limit_lines** (optional): limit number email lines emailed to number
  specified.

**Behavior:**

- Sends plain text emails using SMTP
- Can include output from the unit that triggered it (useful for log/error
  reporting)
- Supports SMTP authentication
- STARTTLS encryption enabled by default
- Works with common email providers (Gmail, SendGrid, Mailgun, etc.)

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
      script: |
        go build -o brun ./cmd/brun
        go test -v
      on_failure:
        - email-failure

  - email:
      name: email-failure
      to:
        - admin@example.com
        - alerts@example.com
      from: brun@example.com
      subject_prefix: "Build Alert"
      smtp_host: smtp.gmail.com
      smtp_port: 587
      smtp_user: brun@example.com
      smtp_password: your-app-password
      smtp_use_tls: true
      include_output: true
```

This will send emails with subjects like:

- `Build Alert: build:success` (on success)
- `Build Alert: build:fail` (on failure)

**Gmail example:**

For Gmail, you need to use an app-specific password:

```yaml
- email:
    name: notify-admin
    to:
      - you@gmail.com
    from: your-app@gmail.com
    subject_prefix: "CI/CD"
    smtp_host: smtp.gmail.com
    smtp_port: 587
    smtp_user: your-app@gmail.com
    smtp_password: your-16-char-app-password
    smtp_use_tls: true
```

### Email Receive Unit (TODO)

This can receive emails to trigger units.

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

## Program lifecycle

BRun traps kill signals and waits for all triggers to complete before exiting.

## Status

This project is in the exploratory phase as we explore various concepts. The
syntax of the BRun file may change as we learn how to better do this.

If you are using BRun, please like this repo and subscribe to release updates.
If there are features you would like, open an issue. This provides motivation to
keep the project going.

Feedback/contributions welcome! Please
[discuss](https://github.com/cbrake/brun/discussions) before implementing
anything major.

See [issues](https://github.com/cbrake/brun/issues) and [ideas](ideas.md) for
future direction.

I have no idea if this works on Windows -- feel free to try and report status.
