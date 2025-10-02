# Simple CI

Simple CI is a tool to run automated builds/tests. Features/goals:

- focus is low level hardware testing
- does not require containers (but may support them in the future)
- simple YAML config format
- designed first to run native
- can run tests/builds on a local workstation outside of traditional

## Config file format

YAML is used for config files leverages the best of Gitlab CI/CD, Drone,
Ansible, and other popular systems.

The system is composed of units. Each unit can trigger additional units. This
allows us to start/sequence operations and create build/test pipelines.

## Units

### Trigger Units

Trigger units watch for various conditions and then trigger when the condition
is met. They are typically used to start other units.

#### System booted

The system booted unit triggers if this is the first time the program has been
run since the system booted.

**How it works:**

The system booted trigger detects boot events by:
1. Reading `/proc/uptime` to calculate the system boot time
2. Comparing this with a stored boot time from the previous run (saved in a state file)
3. Triggering when the boot times differ by more than 10 seconds

The state file is stored at `/var/lib/simpleci/systembooted.state` by default, but can be customized in the configuration.

**Configuration example:**

```yaml
units:
  - system_booted:
      name: boot-trigger
      state_file: /tmp/simpleci-boot.state  # optional, defaults to /var/lib/simpleci/systembooted.state
      trigger:
        - build-unit
        - test-unit
```

When the system booted trigger fires, it will trigger the units listed in the `trigger` array (in this example, `build-unit` and `test-unit`).

#### Git updates

A Git trigger is generated when a Git update is detected in a local workspace.

#### Cron

A Cron trigger unit is configured using the standard Unit cron format.

## Reboot Cycle Test

A reboot cycle test can be enabled by adding the following to your config file:

TODO: create YAML format for this.

The reboot cycle test is typically triggered by a system booted unit.

The reboot cycle counts boot cycles and
