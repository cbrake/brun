package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCountUnit_Run(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)

	unit := NewCountUnit(
		"test-count",
		state,
		[]string{"next-unit"},
		[]string{"error-unit"},
		[]string{"always-unit"},
	)

	if unit.Name() != "test-count" {
		t.Errorf("Expected name 'test-count', got '%s'", unit.Name())
	}

	if unit.Type() != "count" {
		t.Errorf("Expected type 'count', got '%s'", unit.Type())
	}

	ctx := context.Background()

	// First trigger from "build" unit
	unit.SetTriggeringUnit("build")
	if err := unit.Run(ctx); err != nil {
		t.Errorf("First run failed: %v", err)
	}

	// Verify state was saved
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	count, ok := state.Get("test-count", "build")
	if !ok {
		t.Error("Expected count for 'build' to be saved")
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %v", count)
	}

	// Second trigger from "build" unit
	unit.SetTriggeringUnit("build")
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Second run failed: %v", err)
	}

	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	count, ok = state.Get("test-count", "build")
	if !ok {
		t.Error("Expected count for 'build' to be saved")
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %v", count)
	}

	// Trigger from different unit "test"
	unit.SetTriggeringUnit("test")
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Third run failed: %v", err)
	}

	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Should have separate count for "test"
	count, ok = state.Get("test-count", "test")
	if !ok {
		t.Error("Expected count for 'test' to be saved")
	}
	if count != 1 {
		t.Errorf("Expected count 1 for 'test', got %v", count)
	}

	// "build" count should still be 2
	count, ok = state.Get("test-count", "build")
	if !ok {
		t.Error("Expected count for 'build' to be saved")
	}
	if count != 2 {
		t.Errorf("Expected count 2 for 'build', got %v", count)
	}

	onSuccess := unit.OnSuccess()
	if len(onSuccess) != 1 || onSuccess[0] != "next-unit" {
		t.Errorf("Expected on_success [next-unit], got %v", onSuccess)
	}

	onFailure := unit.OnFailure()
	if len(onFailure) != 1 || onFailure[0] != "error-unit" {
		t.Errorf("Expected on_failure [error-unit], got %v", onFailure)
	}

	always := unit.Always()
	if len(always) != 1 || always[0] != "always-unit" {
		t.Errorf("Expected always [always-unit], got %v", always)
	}
}

func TestCountUnit_StateFormat(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)
	unit := NewCountUnit("my-counter", state, nil, nil, nil)

	ctx := context.Background()

	// Trigger from multiple units
	unit.SetTriggeringUnit("unit-a")
	unit.Run(ctx)

	unit.SetTriggeringUnit("unit-a")
	unit.Run(ctx)

	unit.SetTriggeringUnit("unit-b")
	unit.Run(ctx)

	// Read raw state file to verify format
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var stateData map[string]any
	if err := yaml.Unmarshal(data, &stateData); err != nil {
		t.Fatalf("Failed to parse state file: %v", err)
	}

	// Verify structure: my-counter -> unit-a: 2, unit-b: 1
	counterData, ok := stateData["my-counter"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'my-counter' in state")
	}

	if counterData["unit-a"] != 2 {
		t.Errorf("Expected unit-a count to be 2, got %v", counterData["unit-a"])
	}

	if counterData["unit-b"] != 1 {
		t.Errorf("Expected unit-b count to be 1, got %v", counterData["unit-b"])
	}
}

func TestLoadConfig_WithCountUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - count:
      name: trigger-counter
      on_success:
        - notify
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
	if unit.Name() != "trigger-counter" {
		t.Errorf("Expected name 'trigger-counter', got '%s'", unit.Name())
	}

	if unit.Type() != "count" {
		t.Errorf("Expected type 'count', got '%s'", unit.Type())
	}

	countUnit, ok := unit.(*CountUnit)
	if !ok {
		t.Fatal("Unit is not a CountUnit")
	}

	if len(countUnit.onSuccess) != 1 || countUnit.onSuccess[0] != "notify" {
		t.Errorf("Expected on_success [notify], got %v", countUnit.onSuccess)
	}
}
