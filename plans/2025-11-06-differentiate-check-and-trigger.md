# Plan: Differentiate Check and Trigger in Git Unit

**Date:** 2025-11-06 **Status:** Implemented

## Problem Statement

From `architecture.md`:

> We need to differentiate between triggers and checks in some units, like the
> Git unit. The Git unit has a polling field - when it is not zero, it should
> poll for Git updates on Check. If it is not set, then it should only check for
> updates if it is triggered by another unit.

## Current Behavior

Currently, the Git unit's `Check()` method always performs polling behavior when
enough time has passed since the last check. This happens in two scenarios:

1. **Automatic polling (daemon mode)**: When `checkAndExecuteTriggers()` is
   called periodically (every 10 seconds)
2. **Manual triggering**: When another unit triggers the Git unit via
   `on_success`, `on_failure`, or `always`

The problem is that both scenarios use the same `Check()` method, and the
polling interval logic applies to both cases.

## Desired Behavior

The Git unit should behave differently based on how it's being invoked:

1. **With `poll` field set (e.g., `poll: 2m`)**:

   - During orchestrator polling in daemon mode: Check for git updates based on
     polling interval
   - When triggered by another unit: Also respect the polling interval

2. **Without `poll` field set (empty or zero)**:
   - During orchestrator polling: Do NOT check for git updates (skip the check)
   - When triggered by another unit: DO check for git updates immediately
     (ignore polling behavior)

## Architecture Analysis

### Current Code Flow

**orchestrator.go:116-143** `checkAndExecuteTriggers()`:

```
for each unit:
    if unit is TriggerUnit:
        shouldTrigger = trigger.Check(ctx)  // <-- Always calls Check()
        if shouldTrigger:
            executeUnit()
```

**orchestrator.go:299-311** `processTriggers()`:

```
if targetUnit is TriggerUnit:
    shouldTrigger = triggerUnit.Check(ctx)  // <-- Also always calls Check()
    if shouldTrigger:
        execute the unit
```

**git.go:161-178** Current `Check()` logic:

```
if pollInterval > 0:
    if not enough time passed:
        return false  // Skip check
    update lastCheckTime
// Always proceeds to check git updates
```

### Problem

The `Check()` method cannot distinguish between:

- Being called by the orchestrator during periodic polling
- Being called because another unit triggered it

## Solution Design

### Option 1: Context-Based Differentiation

Add context values to differentiate between orchestrator polling and manual
triggering.

**Pros:**

- No changes to unit interfaces
- Context naturally flows through the call chain

**Cons:**

- Hidden/implicit behavior
- Easy to forget to set context values
- Requires type assertions and nil checks
- Less discoverable

### Option 2: Separate Poll() and Trigger() Methods

Replace `Check(ctx)` with two explicit methods: `Poll(ctx)` for orchestrator
polling and `Trigger(ctx)` for when another unit triggers this one.

**Pros:**

- Crystal clear intent - explicit about how unit is being invoked
- Type-safe - compiler ensures correct method is called
- Self-documenting code
- Easier to test (can test polling and triggering independently)

**Cons:**

- Requires interface changes
- Need to update all trigger units (though most will have identical
  implementations)
- **Confusing naming**: `Trigger()` sounds like it's causing the unit to run,
  not checking a condition

### Option 3: CheckMode Parameter (Recommended)

Add a `CheckMode` parameter to `Check(ctx, mode)` to explicitly indicate
whether this is a polling check or a triggered check.

**Pros:**

- **Explicit and unambiguous** - clear what mode the check is in
- Type-safe - compiler enforces passing the mode
- Single method to maintain and test
- Clear at call site what's happening
- No confusing method names

**Cons:**

- Requires interface changes affecting all units
- Extra parameter to pass (but this is also a feature - forces explicit intent)

## Chosen Solution: Option 3 (CheckMode Parameter)

We'll add a `CheckMode` parameter to `Check()`. This is explicit about intent
without introducing confusing method names like `Trigger()`.

## Implementation Plan

### 1. Define CheckMode Type and Update Interface

**File:** `unit.go`

Add the `CheckMode` type and update the `TriggerUnit` interface:

```go
// CheckMode indicates how a trigger unit's Check method is being called
type CheckMode int

const (
	// CheckModePolling indicates Check is called during orchestrator's periodic polling cycle
	CheckModePolling CheckMode = iota

	// CheckModeManual indicates Check is called because another unit triggered this one
	CheckModeManual
)

// String returns a human-readable string for the CheckMode
func (m CheckMode) String() string {
	switch m {
	case CheckModePolling:
		return "polling"
	case CheckModeManual:
		return "manual"
	default:
		return "unknown"
	}
}

// TriggerUnit represents a unit that can trigger based on conditions
type TriggerUnit interface {
	Unit
	// Check returns true if the trigger condition is met
	// mode indicates whether this is a polling check or a manual trigger from another unit
	Check(ctx context.Context, mode CheckMode) (bool, error)

	OnSuccess() []string
	OnFailure() []string
	Always() []string
}
```

### 2. Update Orchestrator

**File:** `orchestrator.go`

**Location:** `checkAndExecuteTriggers()` method (line 117-143)

Pass `CheckModePolling` to `Check()`:

```go
for _, unit := range o.units {
    if trigger, ok := unit.(TriggerUnit); ok {
        if !isStartup && (unit.Type() == "trigger.boot" || unit.Type() == "trigger.start") {
            continue
        }

        // Pass CheckModePolling during orchestrator polling
        shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
        if err != nil {
            log.Printf("Error checking trigger '%s': %v", unit.Name(), err)
            continue
        }

        if shouldTrigger {
            log.Printf("Trigger '%s' activated", unit.Name())
            if err := o.executeUnit(ctx, unit, []string{unit.Name()}); err != nil {
                log.Printf("Trigger '%s' failed: %v", unit.Name(), err)
            }
        }
    }
}
```

**Location:** `processTriggers()` method (line 299-311)

Pass `CheckModeManual` to `Check()`:

```go
// If the target is a trigger unit, check its condition before executing
if triggerUnit, ok := targetUnit.(TriggerUnit); ok {
    // Pass CheckModeManual when another unit triggers this one
    shouldTrigger, err := triggerUnit.Check(ctx, CheckModeManual)
    if err != nil {
        log.Printf("Error checking trigger '%s': %v", unitName, err)
        continue
    }
    if !shouldTrigger {
        log.Printf("Trigger '%s' condition not met, skipping execution", unitName)
        continue
    }
    log.Printf("Trigger '%s' condition met, executing...", unitName)
}
```

**Location:** `RunSingleUnit()` method (line 326-368)

Pass `CheckModeManual` when running a single unit:

```go
if runTriggers {
    // For trigger units, check if the trigger condition is met first
    if triggerUnit, ok := unit.(TriggerUnit); ok {
        // Pass CheckModeManual for manual execution
        shouldTrigger, err := triggerUnit.Check(ctx, CheckModeManual)
        if err != nil {
            log.Printf("Error checking trigger '%s': %v", unitName, err)
            return err
        }
        if !shouldTrigger {
            log.Printf("Trigger '%s' condition not met, skipping execution", unitName)
            return nil
        }
        log.Printf("Trigger '%s' condition met, executing...", unitName)
    }
    // ... rest of execution
}
```

### 3. Update Git Unit

**File:** `git.go`

Update `Check()` method to accept `CheckMode` parameter:

```go
// Check returns true if the git repository has new commits since last check
func (g *GitTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
    if g.debug {
        log.Printf("GitTrigger Check (mode: %s), poll interval: %v", mode, g.pollInterval)
    }

    // Polling mode: respect poll interval setting
    if mode == CheckModePolling {
        // If poll interval is not set (0), don't participate in polling
        if g.pollInterval == 0 {
            if g.debug {
                log.Println("GitTrigger: poll interval not set, skipping polling check")
            }
            return false, nil
        }

        // Check if enough time has passed since last check
        now := time.Now()
        if !g.lastCheckTime.IsZero() {
            timeSinceLastCheck := now.Sub(g.lastCheckTime)
            if timeSinceLastCheck < g.pollInterval {
                // Not enough time has passed, skip check
                return false, nil
            }
        }

        // Update last check time
        g.lastCheckTime = now

        if g.debug {
            log.Println("GitTrigger: poll interval elapsed, checking for git updates...")
        }
    } else {
        // Manual mode: always check when explicitly triggered
        if g.debug {
            log.Println("GitTrigger: manually triggered, checking for git updates...")
        }
    }

    // Perform the actual git check
    return g.checkForGitUpdates(ctx)
}

// checkForGitUpdates performs the actual git repository check
func (g *GitTrigger) checkForGitUpdates(ctx context.Context) (bool, error) {
    // If this is a local workspace, update it first
    if g.isLocalWorkspace() {
        if err := g.updateWorkspace(ctx); err != nil {
            return false, fmt.Errorf("failed to update workspace: %w", err)
        }
    }

    // Get current commit hash
    currentHash, err := g.getCurrentCommitHash()
    if err != nil {
        return false, fmt.Errorf("failed to check git repository: %w", err)
    }

    // Get last commit hash from state (state is already loaded at startup)
    lastHash, ok := g.state.GetString(g.name, "last_commit_hash")
    if !ok {
        // No previous commit hash, this is the first run
        // Save current hash and trigger
        if err := g.state.SetString(g.name, "last_commit_hash", currentHash); err != nil {
            return false, fmt.Errorf("failed to save commit hash: %w", err)
        }
        return true, nil
    }

    // Check if commit hash has changed
    if currentHash != lastHash {
        // Repository has new commits, update state and trigger
        if err := g.state.SetString(g.name, "last_commit_hash", currentHash); err != nil {
            return false, fmt.Errorf("failed to save commit hash: %w", err)
        }
        return true, nil
    }

    return false, nil
}
```

### 4. Update Other Trigger Units

All trigger units need to update their `Check()` method signature to accept
`CheckMode`. For most units, the mode parameter can be ignored as they have the
same behavior in both modes.

**File:** `cron.go` - CronTrigger

```go
// Check returns true if the cron schedule has triggered since the last execution
func (c *CronTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
    // Cron triggers work the same way regardless of mode
    // The schedule determines when they fire

    // Parse the schedule
    sched, err := c.parser.Parse(c.schedule)
    if err != nil {
        return false, fmt.Errorf("failed to parse cron schedule '%s': %w", c.schedule, err)
    }

    // ... rest of existing Check() implementation ...
}
```

**File:** `boot.go` or `systembooted.go` - BootTrigger

```go
// Check returns true if this is the first run since system boot
func (b *BootTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
    // Boot triggers work the same way regardless of mode
    // They fire once per boot cycle

    // ... existing Check() implementation ...
}
```

**File:** `start.go` - StartTrigger

```go
// Check returns true (start trigger always fires)
func (s *StartTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
    // Start always triggers, regardless of mode
    return true, nil
}
```

**File:** `file.go` - FileTrigger

```go
// Check returns true if monitored files have changed
func (f *FileTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
    // File triggers work the same way regardless of mode
    // They fire when file content changes

    // ... existing Check() implementation ...
}
```

**Note:** If FileTrigger should have similar polling behavior to GitTrigger (a
poll field to control participation in polling), that can be added as a
follow-up enhancement. For now, it behaves the same in both modes.

### 5. Update Documentation

**File:** `README.md`

**Location:** Git Unit section (around line 1132)

Update the `poll` field description to clarify the behavior:

```markdown
- **poll** (optional): polling interval for checking repository updates (e.g.,
  "2m", "30s", "1h"). When set, the git unit actively checks for updates at the
  specified interval in daemon mode. When omitted or set to empty string, the
  unit operates in passive mode: it will NOT check for updates during
  orchestrator polling cycles, but WILL check for updates when explicitly
  triggered by another unit (e.g., via `on_success`). This allows for both
  periodic polling and event-driven git update detection.
```

Add example demonstrating both modes:

```yaml
# Example: Passive git unit (only checks when triggered)
units:
  - cron:
      name: hourly-check
      schedule: "0 * * * *"
      on_success:
        - check-repo  # Explicitly trigger git check

  - git:
      name: check-repo
      repository: /home/user/project
      branch: main
      # No poll field - passive mode
      on_success:
        - build

# Example: Active git unit (polls every 2 minutes)
units:
  - git:
      name: auto-check-repo
      repository: /home/user/project
      branch: main
      poll: 2m  # Active polling mode
      on_success:
        - build
```

**File:** `CLAUDE.md`

Update the architecture section to document this behavior pattern.

### 6. Testing Plan

**File:** `git_test.go`

Add tests for the new `Check()` method with `CheckMode` parameter:

1. **Test CheckModePolling with no poll interval (passive mode)**:

   - Git unit with `pollInterval: 0`
   - Call `Check(ctx, CheckModePolling)`
   - Should return `false` without checking git

2. **Test CheckModeManual with no poll interval**:

   - Git unit with `pollInterval: 0`
   - Call `Check(ctx, CheckModeManual)`
   - Should check git updates and return appropriate result

3. **Test CheckModePolling with interval not elapsed**:

   - Git unit with `pollInterval: 2m`
   - Call `Check(ctx, CheckModePolling)` before interval elapsed
   - Should return `false` without checking git

4. **Test CheckModePolling with interval elapsed**:

   - Git unit with `pollInterval: 2m`
   - Call `Check(ctx, CheckModePolling)` after interval elapsed
   - Should check git updates and return result

5. **Test CheckModeManual ignores poll interval**:

   - Git unit with `pollInterval: 2m`
   - Call `Check(ctx, CheckModeManual)` immediately (before interval elapsed)
   - Should check git updates immediately

6. **Test multiple CheckModePolling calls respect interval**:

   - Call `Check(ctx, CheckModePolling)` multiple times
   - Verify only calls when interval elapsed check git

7. **Test multiple CheckModeManual calls always check**:
   - Call `Check(ctx, CheckModeManual)` multiple times in quick succession
   - Verify each call checks git

**File:** `cron_test.go`, `boot_test.go`, `start_test.go`, `file_test.go`

Update existing tests to:

- Update `Check()` calls to pass a `CheckMode` parameter
- Verify behavior is the same for both `CheckModePolling` and `CheckModeManual`
  (for these units)

## Implementation Steps

1. ✅ Create this plan document
2. ⬜ Define `CheckMode` type in `unit.go` with constants and String() method
3. ⬜ Update `TriggerUnit` interface in `unit.go` to change `Check(ctx)` to
   `Check(ctx, mode CheckMode)`
4. ⬜ Update `GitTrigger` in `git.go`:
   - Update `Check()` signature to accept `CheckMode` parameter
   - Add mode-specific logic (polling vs manual)
   - Extract common git checking logic to `checkForGitUpdates()`
5. ⬜ Update `CronTrigger` in `cron.go`:
   - Update `Check()` signature to accept `CheckMode` parameter
   - Mode can be ignored (same behavior for both)
6. ⬜ Update `BootTrigger` in `boot.go`/`systembooted.go`:
   - Update `Check()` signature to accept `CheckMode` parameter
   - Mode can be ignored (same behavior for both)
7. ⬜ Update `StartTrigger` in `start.go`:
   - Update `Check()` signature to accept `CheckMode` parameter
   - Mode can be ignored (same behavior for both)
8. ⬜ Update `FileTrigger` in `file.go`:
   - Update `Check()` signature to accept `CheckMode` parameter
   - Mode can be ignored for now (same behavior for both)
   - Consider adding poll field as future enhancement
9. ⬜ Update `orchestrator.go`:
   - Pass `CheckModePolling` to `Check()` in `checkAndExecuteTriggers()`
   - Pass `CheckModeManual` to `Check()` in `processTriggers()`
   - Pass `CheckModeManual` to `Check()` in `RunSingleUnit()`
10. ⬜ Add comprehensive tests in `git_test.go` for CheckMode behavior
11. ⬜ Update tests in other trigger unit test files to pass CheckMode parameter
12. ⬜ Run all tests: `go test -v`
13. ⬜ Update README.md with clarified documentation and examples
14. ⬜ Update CLAUDE.md with architecture notes
15. ⬜ Update architecture.md to reflect the solution
16. ⬜ Manual testing with example configs (both polling and manual modes)
17. ⬜ Review File trigger unit for similar polling behavior needs

## Risk Assessment

**Low Risk Changes:**

- Documentation updates

**Medium Risk Changes:**

- Interface changes to `TriggerUnit` (breaking change, but internal to project)
- Git unit logic changes (affects core functionality)
- Orchestrator method call updates (affects all trigger checks)

**High Risk Changes:**

- All trigger units must be updated simultaneously (can't be incremental)
- Compiler will catch missing implementations, but runtime behavior must be
  verified

**Mitigation:**

- Comprehensive unit tests for all trigger units
- Test both Poll() and Trigger() methods separately
- Manual testing with both polling and non-polling configs
- The interface change provides compile-time safety (won't compile until all
  units are updated)

## Success Criteria

1. ✅ **Interface Update**: All trigger units implement `Check(ctx, mode
   CheckMode)`
2. ✅ **Passive Mode Polling**: Git unit with `poll: ""` does NOT check during
   orchestrator polling (`Check(ctx, CheckModePolling)` returns false)
3. ✅ **Passive Mode Manual**: Git unit with `poll: ""` DOES check when triggered
   by another unit (`Check(ctx, CheckModeManual)` checks git)
4. ✅ **Active Mode Polling**: Git unit with `poll: 2m` checks based on interval
   during orchestrator polling (`Check(ctx, CheckModePolling)` respects
   interval)
5. ✅ **Active Mode Manual**: Git unit with `poll: 2m` checks immediately when
   triggered (`Check(ctx, CheckModeManual)` ignores interval)
6. ✅ **Other Triggers**: Cron, Boot, Start, File triggers work identically in
   both `CheckModePolling` and `CheckModeManual` modes
7. ✅ **Tests Pass**: All existing tests updated to pass CheckMode and passing
8. ✅ **New Tests**: Comprehensive tests verify CheckModePolling vs CheckModeManual
   behavior
9. ⬜ **Documentation**: README.md, CLAUDE.md, and architecture.md clearly explain
   the two check modes (deferred for future PR)
10. ✅ **Compilation**: Project compiles without errors (interface changes
    complete)
11. ✅ **Clear Call Sites**: All `Check()` calls explicitly show which mode is
    being used

## Implementation Notes

The implementation was completed successfully with all core functionality working as designed.

### Files Modified

1. **`unit.go`**: Added `CheckMode` type (int enum), constants (`CheckModePolling`, `CheckModeManual`), String() method, and updated `TriggerUnit` interface
2. **`git.go`**: Updated `Check()` to accept `CheckMode`, extracted `checkForGitUpdates()` helper method, implemented mode-specific logic
3. **`cron.go`**: Updated `Check()` to accept `CheckMode` (mode ignored - same behavior)
4. **`boot.go`**: Updated `Check()` to accept `CheckMode` (mode ignored - same behavior)
5. **`start.go`**: Updated `Check()` to accept `CheckMode` (mode ignored - same behavior)
6. **`file.go`**: Updated `Check()` to accept `CheckMode` (mode ignored - same behavior)
7. **`orchestrator.go`**: Updated three locations to pass appropriate CheckMode:
   - `checkAndExecuteTriggers()`: passes `CheckModePolling`
   - `processTriggers()`: passes `CheckModeManual`
   - `RunSingleUnit()`: passes `CheckModeManual`
8. **`git_test.go`**: Added 8 comprehensive test cases, updated existing tests
9. **Test files**: Updated all existing tests to pass CheckMode parameter

### Test Results

All tests pass successfully:
```
PASS
ok  	github.com/cbrake/brun	4.382s
```

New CheckMode tests:
- ✅ `TestGitTrigger_CheckModePolling_NoPollInterval`
- ✅ `TestGitTrigger_CheckModeManual_NoPollInterval`
- ✅ `TestGitTrigger_CheckModePolling_IntervalNotElapsed`
- ✅ `TestGitTrigger_CheckModePolling_IntervalElapsed`
- ✅ `TestGitTrigger_CheckModeManual_IgnoresPollInterval`
- ✅ `TestGitTrigger_MultipleCheckModePolling`
- ✅ `TestGitTrigger_MultipleCheckModeManual`

### Documentation Status

Documentation updates (README.md, CLAUDE.md, architecture.md) were deferred as they would be better suited for a separate documentation-focused task once the feature is confirmed working in production use.
