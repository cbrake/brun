package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBootDetector_GetBootTime(t *testing.T) {
	bd := NewBootDetector("/tmp/test-state")
	bootTime, err := bd.GetBootTime()
	if err != nil {
		t.Fatalf("GetBootTime failed: %v", err)
	}

	// Boot time should be in the past
	if bootTime.After(time.Now()) {
		t.Errorf("Boot time is in the future: %v", bootTime)
	}

	// Boot time should be reasonable (not more than 30 days ago)
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	if bootTime.Before(thirtyDaysAgo) {
		t.Errorf("Boot time is too far in the past: %v", bootTime)
	}
}

func TestBootDetector_IsFirstRunSinceBoot(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "boot.state")

	bd := NewBootDetector(stateFile)

	// First call should return true (first run ever)
	firstRun, err := bd.IsFirstRunSinceBoot()
	if err != nil {
		t.Fatalf("IsFirstRunSinceBoot failed: %v", err)
	}
	if !firstRun {
		t.Error("Expected first run to be true")
	}

	// Second call should return false (same boot session)
	secondRun, err := bd.IsFirstRunSinceBoot()
	if err != nil {
		t.Fatalf("IsFirstRunSinceBoot failed on second call: %v", err)
	}
	if secondRun {
		t.Error("Expected second run to be false")
	}

	// Verify state file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file was not created")
	}
}

func TestBootDetector_InvalidStateFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "boot.state")

	// Write invalid data to state file
	if err := os.WriteFile(stateFile, []byte("invalid-time-format"), 0644); err != nil {
		t.Fatalf("Failed to write test state file: %v", err)
	}

	bd := NewBootDetector(stateFile)

	// Should treat as first run and overwrite invalid state
	firstRun, err := bd.IsFirstRunSinceBoot()
	if err != nil {
		t.Fatalf("IsFirstRunSinceBoot failed: %v", err)
	}
	if !firstRun {
		t.Error("Expected first run to be true with invalid state file")
	}

	// Verify state file now has valid data
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}
	_, err = time.Parse(time.RFC3339, string(data))
	if err != nil {
		t.Errorf("State file still contains invalid time format: %v", err)
	}
}

func TestBootTrigger_Check(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)
	trigger := NewBootTrigger("test-boot", state, []string{"unit1"}, nil, nil)

	ctx := context.Background()

	// First check should trigger
	triggered, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !triggered {
		t.Error("Expected trigger to activate on first check")
	}

	// Second check should not trigger
	triggered, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed on second call: %v", err)
	}
	if triggered {
		t.Error("Expected trigger to not activate on second check")
	}
}

func TestBootTrigger_Run(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)
	trigger := NewBootTrigger("test-boot", state, []string{"unit1", "unit2"}, []string{"fail-unit"}, []string{"always-unit"})

	if trigger.Name() != "test-boot" {
		t.Errorf("Expected name 'test-boot', got '%s'", trigger.Name())
	}

	if trigger.Type() != "trigger.boot" {
		t.Errorf("Expected type 'trigger.boot', got '%s'", trigger.Type())
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 2 || onSuccess[0] != "unit1" || onSuccess[1] != "unit2" {
		t.Errorf("Expected on_success units [unit1, unit2], got %v", onSuccess)
	}

	onFailure := trigger.OnFailure()
	if len(onFailure) != 1 || onFailure[0] != "fail-unit" {
		t.Errorf("Expected on_failure units [fail-unit], got %v", onFailure)
	}

	always := trigger.Always()
	if len(always) != 1 || always[0] != "always-unit" {
		t.Errorf("Expected always units [always-unit], got %v", always)
	}

	ctx := context.Background()
	if err := trigger.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestBootTrigger_StateFilePath(t *testing.T) {
	stateFile := "/tmp/test-state.yaml"
	state := NewState(stateFile)
	trigger := NewBootTrigger("test", state, nil, nil, nil)

	// Should use the provided state file path
	if trigger.state.filePath != stateFile {
		t.Errorf("Expected state file '%s', got '%s'", stateFile, trigger.state.filePath)
	}
}
