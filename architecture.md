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
