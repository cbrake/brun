Trigger/Input Features

- Webhook unit - HTTP endpoint to trigger workflows (like GitHub webhooks,
  GitLab CI, Drone)
- MQTT unit - Subscribe to MQTT topics to trigger workflows (IoT/embedded focus)
- System signal unit - Trigger on SIGUSR1/SIGUSR2 for manual triggering without
  restarting
- File watch directories - Monitor entire directories for any changes, not just
  specific patterns
- Database trigger - Monitor database changes (common in automation tools)

Execution/Action Features

- Docker/Podman unit - Run commands in containers (optional, since bare-OS is
  priority)
- HTTP request unit - Make HTTP calls, POST data, check endpoints (like
  Ansible's uri module)
- SSH/Remote unit - Execute commands on remote machines (like Ansible)
- File operations unit - Copy, move, delete files (like Ansible file module)
- Template unit - Generate config files from templates (like Ansible template)
- Archive unit - Create tar/zip archives for backups

Notification/Output Features

- Slack/Discord/Teams notifications - Modern team communication
- Webhook POST - Send data to arbitrary HTTP endpoints
- SMS notifications - Via Twilio or similar (critical alerts)
- Telegram bot - Popular for server notifications

Control Flow Features

- Conditional execution - Run units based on conditions (env vars, file
  existence, etc.)
- Parallel execution - Run multiple units simultaneously instead of sequentially
- Retry logic - Automatically retry failed units with backoff
- Timeout per unit - Already have for run unit, extend to others
- Variables/environment - Pass data between units, environment substitution

State/Data Features

- Artifact storage - Save build outputs, logs, test results
- Metrics collection - Track execution times, success rates (like Prometheus)
- State queries - CLI commands to inspect current state without running
- State backup/restore - Automatic state backups

Configuration Features

- Include/import configs - Split large configs into multiple files
- Config validation - Dry-run mode to check config without executing
- Unit templates/macros - Reusable unit definitions (like Ansible roles)
- Environment-specific configs - dev/staging/prod variants

Debugging/Observability Features

- Interactive mode - Prompt before running units
- Verbose logging levels - Control log verbosity
- Unit status dashboard - Web UI or TUI to see what's running
- Execution history - Store past executions with timestamps, outputs
- Profiling - See which units take longest

Security Features

- Secret management - Integration with vault tools, encrypted secrets in config
- User permissions - Different users can run different units
- Audit logging - Who ran what, when

Most Valuable for BRun's Focus (Embedded/Bare-OS Testing)

Given BRun's focus on embedded and bare-OS testing, these would be most
valuable:

1. System signal unit - Easy manual triggering
2. SSH/Remote unit - Deploy to target devices
3. MQTT unit - Very common in embedded/IoT
4. Retry logic - Hardware can be flaky
5. Conditional execution - Different behavior per board/config
6. Artifact storage - Save test results, build outputs
7. Variables/environment - Board-specific configuration
8. Metrics collection - Track test pass rates, boot times

The webhook unit would also be particularly useful for integration with version
control systems and CI/CD pipelines.
