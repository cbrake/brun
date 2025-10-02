package simpleci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStartTrigger_Check(t *testing.T) {
	trigger := NewStartTrigger(
		"test-start",
		[]string{"build"},
		[]string{"error"},
		[]string{"log"},
	)

	if trigger.Name() != "test-start" {
		t.Errorf("Expected name 'test-start', got '%s'", trigger.Name())
	}

	if trigger.Type() != "trigger.start" {
		t.Errorf("Expected type 'trigger.start', got '%s'", trigger.Type())
	}

	ctx := context.Background()
	shouldTrigger, err := trigger.Check(ctx)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if !shouldTrigger {
		t.Error("Start trigger should always return true")
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 1 || onSuccess[0] != "build" {
		t.Errorf("Expected on_success [build], got %v", onSuccess)
	}

	onFailure := trigger.OnFailure()
	if len(onFailure) != 1 || onFailure[0] != "error" {
		t.Errorf("Expected on_failure [error], got %v", onFailure)
	}

	always := trigger.Always()
	if len(always) != 1 || always[0] != "log" {
		t.Errorf("Expected always [log], got %v", always)
	}
}

func TestStartTrigger_Run(t *testing.T) {
	trigger := NewStartTrigger(
		"test-start",
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	if err := trigger.Run(ctx); err != nil {
		t.Errorf("Run failed: %v", err)
	}
}

func TestStartTrigger_AlwaysTriggers(t *testing.T) {
	trigger := NewStartTrigger(
		"test-start",
		nil,
		nil,
		nil,
	)

	ctx := context.Background()

	// Check multiple times to ensure it always triggers
	for i := 0; i < 5; i++ {
		shouldTrigger, err := trigger.Check(ctx)
		if err != nil {
			t.Errorf("Check failed on iteration %d: %v", i, err)
		}
		if !shouldTrigger {
			t.Errorf("Start trigger should always return true on iteration %d", i)
		}
	}
}

func TestLoadConfig_WithStartUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - start:
      name: start-trigger
      on_success:
        - build
        - test
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
	if unit.Name() != "start-trigger" {
		t.Errorf("Expected name 'start-trigger', got '%s'", unit.Name())
	}

	if unit.Type() != "trigger.start" {
		t.Errorf("Expected type 'trigger.start', got '%s'", unit.Type())
	}

	trigger, ok := unit.(TriggerUnit)
	if !ok {
		t.Fatal("Unit is not a TriggerUnit")
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 2 || onSuccess[0] != "build" || onSuccess[1] != "test" {
		t.Errorf("Expected on_success [build, test], got %v", onSuccess)
	}
}
