package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCronTrigger_Check(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	// Create a cron trigger that runs every minute
	trigger := NewCronTrigger(
		"test-cron",
		"* * * * *",
		state,
		[]string{"next-unit"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check - should trigger since no last execution
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected first check to trigger")
	}

	// Verify state was saved
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	lastExec, ok := state.GetString("test-cron", "last_execution")
	if !ok {
		t.Error("Expected last_execution to be saved")
	}

	// Parse and verify it's a valid time
	execTime, err := time.Parse(time.RFC3339, lastExec)
	if err != nil {
		t.Fatalf("Failed to parse execution time: %v", err)
	}

	// With the fix, we now save the scheduled time (minute boundary) not current time
	// So execution time could be up to 60 seconds in the past
	// Just verify it's not too far in the past (within 2 minutes to be safe)
	if time.Since(execTime) > 2*time.Minute {
		t.Errorf("Execution time should be recent (within 2 minutes), but was %v ago", time.Since(execTime))
	}

	// Verify it's on a minute boundary (seconds = 0) since we save scheduled time
	if execTime.Second() != 0 {
		t.Errorf("Expected execution time to be on minute boundary (seconds=0), got %d seconds", execTime.Second())
	}

	// Immediate second check - should not trigger (same minute)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Second check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected second check not to trigger (same minute)")
	}
}

func TestCronTrigger_InvalidSchedule(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	// Create a cron trigger with invalid schedule
	trigger := NewCronTrigger(
		"test-cron-invalid",
		"invalid schedule",
		state,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()

	// Check should fail with invalid schedule
	_, err := trigger.Check(ctx, CheckModePolling)
	if err == nil {
		t.Error("Expected error for invalid schedule")
	}
}

func TestCronTrigger_Run(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	trigger := NewCronTrigger(
		"test-cron-run",
		"* * * * *",
		state,
		[]string{"next-unit"},
		nil,
		nil,
	)

	if trigger.Name() != "test-cron-run" {
		t.Errorf("Expected name 'test-cron-run', got '%s'", trigger.Name())
	}

	if trigger.Type() != "trigger.cron" {
		t.Errorf("Expected type 'trigger.cron', got '%s'", trigger.Type())
	}

	ctx := context.Background()
	if err := trigger.Run(ctx); err != nil {
		t.Errorf("Run failed: %v", err)
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 1 || onSuccess[0] != "next-unit" {
		t.Errorf("Expected on_success [next-unit], got %v", onSuccess)
	}
}

func TestLoadConfig_WithCronUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - cron:
      name: test-cron-config
      schedule: "*/5 * * * *"
      on_success:
        - build
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	units, err := config.CreateUnits()
	if err != nil {
		t.Fatalf("CreateUnits failed: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(units))
	}

	unit := units[0]
	if unit.Name() != "test-cron-config" {
		t.Errorf("Expected name 'test-cron-config', got '%s'", unit.Name())
	}

	if unit.Type() != "trigger.cron" {
		t.Errorf("Expected type 'trigger.cron', got '%s'", unit.Type())
	}

	cronUnit, ok := unit.(*CronTrigger)
	if !ok {
		t.Fatal("Unit is not a CronTrigger")
	}

	if cronUnit.schedule != "*/5 * * * *" {
		t.Errorf("Expected schedule '*/5 * * * *', got '%s'", cronUnit.schedule)
	}

	if len(cronUnit.onSuccess) != 1 || cronUnit.onSuccess[0] != "build" {
		t.Errorf("Expected on_success [build], got %v", cronUnit.onSuccess)
	}
}

func TestCreateUnits_CronMissingSchedule(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Cron: &CronConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// Schedule is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing schedule")
	}
}

func TestCronTrigger_SkipMissedRun(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	// Create a cron trigger that runs daily at midnight
	trigger := NewCronTrigger(
		"test-cron-skip",
		"0 0 * * *",
		state,
		[]string{"next-unit"},
		nil,
		nil,
	)

	ctx := context.Background()

	// Simulate that the last execution was 2 days ago at 11 PM
	// This means we missed yesterday's midnight run
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	if err := state.SetString("test-cron-skip", "last_execution", twoDaysAgo.Format(time.RFC3339)); err != nil {
		t.Fatalf("Failed to set last_execution: %v", err)
	}

	// First check - should NOT trigger because we missed the scheduled time
	// (we're way past the tolerance window)
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected not to trigger for missed run (outside tolerance window)")
	}

	// Verify state was updated to current time (skipped the missed run)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	lastExec, ok := state.GetString("test-cron-skip", "last_execution")
	if !ok {
		t.Error("Expected last_execution to be updated")
	}

	// Parse and verify it's recent (within last 5 seconds)
	execTime, err := time.Parse(time.RFC3339, lastExec)
	if err != nil {
		t.Fatalf("Failed to parse execution time: %v", err)
	}

	if time.Since(execTime) > 5*time.Second {
		t.Error("Execution time should be very recent (missed run was skipped and time updated to now)")
	}
}

func TestCronTrigger_WithinToleranceWindow(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	// Create a cron trigger that runs every minute
	trigger := NewCronTrigger(
		"test-cron-tolerance",
		"* * * * *",
		state,
		[]string{"next-unit"},
		nil,
		nil,
	)

	ctx := context.Background()

	// Calculate a last execution time that guarantees the next scheduled time
	// is within the tolerance window. We find the most recent minute boundary
	// and set last execution to 30 seconds before that.
	// This ensures next scheduled time is the recent minute boundary,
	// which is guaranteed to be less than 60 seconds ago.
	now := time.Now()
	// Truncate to the most recent minute boundary
	recentMinuteBoundary := now.Truncate(time.Minute)
	// Set last execution to 30 seconds before the recent minute boundary
	// For a * * * * * schedule, the next scheduled time after this will be
	// the recent minute boundary, which is at most 59 seconds ago
	lastExec := recentMinuteBoundary.Add(-30 * time.Second)
	if err := state.SetString("test-cron-tolerance", "last_execution", lastExec.Format(time.RFC3339)); err != nil {
		t.Fatalf("Failed to set last_execution: %v", err)
	}

	// Check - should trigger because we're within the tolerance window
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected to trigger when within tolerance window of scheduled time")
	}
}

func TestCronTrigger_NoDoubleTrigger(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	// Create a cron trigger that runs every minute
	// This ensures the test works regardless of when it's run
	trigger := NewCronTrigger(
		"test-cron-double",
		"* * * * *",
		state,
		[]string{"next-unit"},
		nil,
		nil,
	)

	ctx := context.Background()

	// Clear the state to simulate first run
	state.data = make(map[string]any)
	if err := state.Save(); err != nil {
		t.Fatalf("Failed to clear state: %v", err)
	}

	// First check - should trigger because no previous execution and we're past a minute boundary
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("First check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected first check to trigger")
	}

	// Reload state to ensure we're reading what was saved
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to reload state: %v", err)
	}

	// Second check immediately after (simulating orchestrator check 10 seconds later)
	// This should NOT trigger because we already handled the current minute's run
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Second check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected second check NOT to trigger (this would be the double-trigger bug)")
	}

	// Verify that the last_execution time is set to the scheduled time (minute boundary)
	// not to "now" which would be a few seconds after the minute boundary
	lastExecStr, ok := state.GetString("test-cron-double", "last_execution")
	if !ok {
		t.Fatal("Expected last_execution to be saved")
	}

	lastExecTime, err := time.Parse(time.RFC3339, lastExecStr)
	if err != nil {
		t.Fatalf("Failed to parse last_execution: %v", err)
	}

	// The saved time should be on a minute boundary (seconds = 0) since we save the scheduled time
	// Check that seconds are 0 (indicating we saved the scheduled time, not current time)
	if lastExecTime.Second() != 0 {
		t.Errorf("Expected last_execution to be saved with 0 seconds (scheduled time), got %d seconds", lastExecTime.Second())
	}
}
