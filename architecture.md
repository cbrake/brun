# BRun Architecture

This document details some of the inner workings of BRun.

## Units

Each unit is defined by a:

- Unit type
- Unit config type

Each Unit has its own type, but does not store config -- that is stored in the
orchestrator and passed in at unit execution time.

The Unit `Run()` method returns:

- nop
- failure
- success

The Orchestrator then uses these return codes to execute triggers. nop means not
triggers (except `always` will be executed).

## Orchestrator

The Orchestrator is responsible for:

- storing the unit config and expanding templates at unit check and run time.
- checking if units need to execute
- executing the triggers

**CheckMode Differentiation**: Trigger units differentiate between orchestrator
polling and explicit triggering via a `CheckMode` parameter passed to
`Check(ctx, mode)`:

- `CheckModePolling`: Orchestrator's periodic check cycle (every 10s in daemon
  mode)
- `CheckModeManual`: Another unit explicitly triggered this one via
  `on_success`, `on_failure`, or `always`

Git units use this to implement two behaviors:

- With `poll` field: Participates in periodic polling based on interval
- Without `poll` field: Passive - only checks when explicitly triggered by other
  units

This enables event-driven workflows where git checks happen on-demand (e.g., a
cron unit triggers a git check) without constant polling overhead.
