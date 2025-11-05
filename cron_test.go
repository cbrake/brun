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
	shouldTrigger, err := trigger.Check(ctx)
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

	// Parse and verify it's recent
	execTime, err := time.Parse(time.RFC3339, lastExec)
	if err != nil {
		t.Fatalf("Failed to parse execution time: %v", err)
	}

	if time.Since(execTime) > 5*time.Second {
		t.Error("Execution time should be very recent")
	}

	// Immediate second check - should not trigger (same minute)
	shouldTrigger, err = trigger.Check(ctx)
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
	_, err := trigger.Check(ctx)
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
	shouldTrigger, err := trigger.Check(ctx)
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

	// Set last execution to 70 seconds ago
	// For a every-minute cron, next scheduled time will be ~10 seconds ago
	// (the most recent minute boundary), which is within the 60-second tolerance
	seventySecondsAgo := time.Now().Add(-70 * time.Second)
	if err := state.SetString("test-cron-tolerance", "last_execution", seventySecondsAgo.Format(time.RFC3339)); err != nil {
		t.Fatalf("Failed to set last_execution: %v", err)
	}

	// Check - should trigger because we're within the tolerance window
	shouldTrigger, err := trigger.Check(ctx)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected to trigger when within tolerance window of scheduled time")
	}
}
