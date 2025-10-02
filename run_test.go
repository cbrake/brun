package metalci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunUnit_Success(t *testing.T) {
	unit := NewRunUnit(
		"test-run",
		"echo 'Hello World'",
		"",
		[]string{"next-unit"},
		[]string{"error-unit"},
		[]string{"always-unit"},
	)

	if unit.Name() != "test-run" {
		t.Errorf("Expected name 'test-run', got '%s'", unit.Name())
	}

	if unit.Type() != "run" {
		t.Errorf("Expected type 'run', got '%s'", unit.Type())
	}

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
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

func TestRunUnit_Failure(t *testing.T) {
	unit := NewRunUnit(
		"test-run-fail",
		"exit 1",
		"",
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	err := unit.Run(ctx)
	if err == nil {
		t.Error("Expected error for failing script")
	}
}

func TestRunUnit_WithDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	unit := NewRunUnit(
		"test-run-dir",
		"echo 'test' > test.txt",
		tempDir,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Verify file was created in the correct directory
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("Expected test.txt to be created in %s", tempDir)
	}
}

func TestRunUnit_MultilineScript(t *testing.T) {
	tempDir := t.TempDir()

	script := `
echo "line 1"
echo "line 2"
echo "line 3"
`

	unit := NewRunUnit(
		"test-multiline",
		script,
		tempDir,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestLoadConfig_WithRunUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - run:
      name: build
      script: |
        echo "building"
        go build
      directory: /tmp
      on_success:
        - test
      on_failure:
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
	if unit.Name() != "build" {
		t.Errorf("Expected name 'build', got '%s'", unit.Name())
	}

	if unit.Type() != "run" {
		t.Errorf("Expected type 'run', got '%s'", unit.Type())
	}

	runUnit, ok := unit.(*RunUnit)
	if !ok {
		t.Fatal("Unit is not a RunUnit")
	}

	if runUnit.directory != "/tmp" {
		t.Errorf("Expected directory '/tmp', got '%s'", runUnit.directory)
	}

	if len(runUnit.onSuccess) != 1 || runUnit.onSuccess[0] != "test" {
		t.Errorf("Expected on_success [test], got %v", runUnit.onSuccess)
	}
}

func TestCreateUnits_RunMissingScript(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Run: &RunConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// Script is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing script")
	}
}
