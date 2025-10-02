package simpleci

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configContent := `units:
  - system_booted:
      name: boot-trigger
      trigger:
        - build-unit
        - test-unit
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(config.Units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(config.Units))
	}

	if config.Units[0].SystemBooted == nil {
		t.Fatal("Expected SystemBooted config to be non-nil")
	}

	bootConfig := config.Units[0].SystemBooted
	if bootConfig.Name != "boot-trigger" {
		t.Errorf("Expected name 'boot-trigger', got '%s'", bootConfig.Name)
	}

	if len(bootConfig.Trigger) != 2 {
		t.Fatalf("Expected 2 trigger units, got %d", len(bootConfig.Trigger))
	}

	if bootConfig.Trigger[0] != "build-unit" || bootConfig.Trigger[1] != "test-unit" {
		t.Errorf("Unexpected trigger units: %v", bootConfig.Trigger)
	}
}

func TestCreateUnits(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "boot.state")

	config := &Config{
		Units: []UnitConfigWrapper{
			{
				SystemBooted: &SystemBootedConfig{
					UnitConfig: UnitConfig{
						Name:    "boot-trigger",
						Trigger: []string{"build", "test"},
					},
					StateFile: stateFile,
				},
			},
		},
	}

	units, err := config.CreateUnits()
	if err != nil {
		t.Fatalf("CreateUnits failed: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(units))
	}

	unit := units[0]
	if unit.Name() != "boot-trigger" {
		t.Errorf("Expected name 'boot-trigger', got '%s'", unit.Name())
	}

	if unit.Type() != "trigger.systembooted" {
		t.Errorf("Expected type 'trigger.systembooted', got '%s'", unit.Type())
	}

	trigger, ok := unit.(TriggerUnit)
	if !ok {
		t.Fatal("Unit is not a TriggerUnit")
	}

	triggerUnits := trigger.OnTrigger()
	if len(triggerUnits) != 2 || triggerUnits[0] != "build" || triggerUnits[1] != "test" {
		t.Errorf("Expected trigger units [build, test], got %v", triggerUnits)
	}
}

func TestCreateUnits_MissingName(t *testing.T) {
	config := &Config{
		Units: []UnitConfigWrapper{
			{
				SystemBooted: &SystemBootedConfig{
					UnitConfig: UnitConfig{
						Trigger: []string{"build"},
					},
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing name")
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	if err := os.WriteFile(configFile, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
