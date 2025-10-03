package brun

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := fmt.Sprintf(`config:
  state_location: %s

units:
  - boot:
      name: boot-trigger
      on_success:
        - build-unit
        - test-unit
      on_failure:
        - notify-admin
      always:
        - log-unit
`, stateFile)

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

	if config.Units[0].Boot == nil {
		t.Fatal("Expected Boot config to be non-nil")
	}

	bootConfig := config.Units[0].Boot
	if bootConfig.Name != "boot-trigger" {
		t.Errorf("Expected name 'boot-trigger', got '%s'", bootConfig.Name)
	}

	if len(bootConfig.OnSuccess) != 2 {
		t.Fatalf("Expected 2 on_success units, got %d", len(bootConfig.OnSuccess))
	}

	if bootConfig.OnSuccess[0] != "build-unit" || bootConfig.OnSuccess[1] != "test-unit" {
		t.Errorf("Unexpected on_success units: %v", bootConfig.OnSuccess)
	}

	if len(bootConfig.OnFailure) != 1 || bootConfig.OnFailure[0] != "notify-admin" {
		t.Errorf("Unexpected on_failure units: %v", bootConfig.OnFailure)
	}

	if len(bootConfig.Always) != 1 || bootConfig.Always[0] != "log-unit" {
		t.Errorf("Unexpected always units: %v", bootConfig.Always)
	}
}

func TestCreateUnits(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Boot: &BootConfig{
					UnitConfig: UnitConfig{
						Name:      "boot-trigger",
						OnSuccess: []string{"build", "test"},
						OnFailure: []string{"cleanup"},
						Always:    []string{"log"},
					},
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

	if unit.Type() != "trigger.boot" {
		t.Errorf("Expected type 'trigger.boot', got '%s'", unit.Type())
	}

	trigger, ok := unit.(TriggerUnit)
	if !ok {
		t.Fatal("Unit is not a TriggerUnit")
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 2 || onSuccess[0] != "build" || onSuccess[1] != "test" {
		t.Errorf("Expected on_success units [build, test], got %v", onSuccess)
	}

	onFailure := trigger.OnFailure()
	if len(onFailure) != 1 || onFailure[0] != "cleanup" {
		t.Errorf("Expected on_failure units [cleanup], got %v", onFailure)
	}

	always := trigger.Always()
	if len(always) != 1 || always[0] != "log" {
		t.Errorf("Expected always units [log], got %v", always)
	}
}

func TestCreateUnits_MissingName(t *testing.T) {
	config := &Config{
		Units: []UnitConfigWrapper{
			{
				Boot: &BootConfig{
					UnitConfig: UnitConfig{
						OnSuccess: []string{"build"},
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

func TestCreateUnits_MissingStateLocation(t *testing.T) {
	config := &Config{
		Units: []UnitConfigWrapper{
			{
				Boot: &BootConfig{
					UnitConfig: UnitConfig{
						Name:      "boot-trigger",
						OnSuccess: []string{"build"},
					},
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing state_location")
	}
}
