package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunUnit_Success(t *testing.T) {
	unit := NewRunUnit(
		"test-run",
		"echo 'Hello World'",
		"",
		0,
		"",
		false,
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
		0,
		"",
		false,
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
		0,
		"",
		false,
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
		0,
		"",
		false,
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

func TestRunUnit_WithTimeout(t *testing.T) {
	// Test that a task times out correctly
	unit := NewRunUnit(
		"test-timeout",
		"sleep 5",
		"",
		1*time.Second,
		"",
		false,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	err := unit.Run(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if err != nil && err.Error() != "task timed out after 1s" {
		t.Errorf("Expected timeout error message, got: %v", err)
	}
}

func TestLoadConfig_WithTimeout(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - run:
      name: quick-task
      script: echo "done"
      timeout: 30s
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

	runUnit, ok := units[0].(*RunUnit)
	if !ok {
		t.Fatal("Unit is not a RunUnit")
	}

	if runUnit.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", runUnit.timeout)
	}
}

func TestLoadConfig_InvalidTimeout(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - run:
      name: bad-timeout
      script: echo "test"
      timeout: invalid
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	_, err = config.CreateUnits()
	if err == nil {
		t.Error("Expected error for invalid timeout format")
	}
}

func TestRunUnit_WithShell(t *testing.T) {
	// Test with bash shell
	unit := NewRunUnit(
		"test-bash",
		"echo 'Hello from bash'",
		"",
		0,
		"bash",
		false,
		nil,
		nil,
		nil,
	)

	if unit.shell != "bash" {
		t.Errorf("Expected shell 'bash', got '%s'", unit.shell)
	}

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestRunUnit_DefaultShell(t *testing.T) {
	// Test that default shell is 'sh' when not specified
	unit := NewRunUnit(
		"test-default-shell",
		"echo 'Hello'",
		"",
		0,
		"",
		false,
		nil,
		nil,
		nil,
	)

	if unit.shell != "sh" {
		t.Errorf("Expected default shell 'sh', got '%s'", unit.shell)
	}
}

func TestLoadConfig_WithShell(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - run:
      name: bash-task
      script: echo "running with bash"
      shell: bash
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

	runUnit, ok := units[0].(*RunUnit)
	if !ok {
		t.Fatal("Unit is not a RunUnit")
	}

	if runUnit.shell != "bash" {
		t.Errorf("Expected shell 'bash', got '%s'", runUnit.shell)
	}
}

func TestRunUnit_WithPTY(t *testing.T) {
	// Test with PTY enabled
	unit := NewRunUnit(
		"test-pty",
		"echo 'Hello with PTY'",
		"",
		0,
		"bash",
		true,
		nil,
		nil,
		nil,
	)

	if !unit.usePTY {
		t.Error("Expected usePTY to be true")
	}

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestLoadConfig_WithPTY(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - run:
      name: pty-task
      script: echo "running with PTY"
      shell: bash
      use_pty: true
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

	runUnit, ok := units[0].(*RunUnit)
	if !ok {
		t.Fatal("Unit is not a RunUnit")
	}

	if !runUnit.usePTY {
		t.Error("Expected usePTY to be true")
	}
}
