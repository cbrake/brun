package metalci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogUnit_Run(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	unit := NewLogUnit(
		"test-log",
		logFile,
		[]string{"next-unit"},
		[]string{"error-unit"},
		[]string{"always-unit"},
	)

	if unit.Name() != "test-log" {
		t.Errorf("Expected name 'test-log', got '%s'", unit.Name())
	}

	if unit.Type() != "log" {
		t.Errorf("Expected type 'log', got '%s'", unit.Type())
	}

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Verify log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Expected log file to be created at %s", logFile)
	}

	// Verify file contains log entry
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "unknown") {
		t.Errorf("Expected log file to contain 'unknown' (no triggering unit set), got: %s", contentStr)
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

func TestLogUnit_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "nested", "dir", "test.log")

	unit := NewLogUnit(
		"test-log-nested",
		logFile,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Verify nested directories were created
	if _, err := os.Stat(filepath.Dir(logFile)); err != nil {
		t.Errorf("Expected nested directories to be created")
	}

	// Verify log file was created
	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Expected log file to be created at %s", logFile)
	}
}

func TestLogUnit_Appends(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	unit := NewLogUnit(
		"test-log-append",
		logFile,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()

	// Run first time
	if err := unit.Run(ctx); err != nil {
		t.Errorf("First run failed: %v", err)
	}

	// Run second time
	if err := unit.Run(ctx); err != nil {
		t.Errorf("Second run failed: %v", err)
	}

	// Verify file contains multiple entries
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	count := strings.Count(contentStr, "=== Unit")
	if count != 2 {
		t.Errorf("Expected 2 log entries, found %d. Content: %s", count, contentStr)
	}
}

func TestLoadConfig_WithLogUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")
	logFile := filepath.Join(tempDir, "app.log")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - log:
      name: error-log
      file: ` + logFile + `
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
	if unit.Name() != "error-log" {
		t.Errorf("Expected name 'error-log', got '%s'", unit.Name())
	}

	if unit.Type() != "log" {
		t.Errorf("Expected type 'log', got '%s'", unit.Type())
	}

	logUnit, ok := unit.(*LogUnit)
	if !ok {
		t.Fatal("Unit is not a LogUnit")
	}

	if logUnit.file != logFile {
		t.Errorf("Expected file '%s', got '%s'", logFile, logUnit.file)
	}

	if len(logUnit.onSuccess) != 1 || logUnit.onSuccess[0] != "notify" {
		t.Errorf("Expected on_success [notify], got %v", logUnit.onSuccess)
	}
}

func TestCreateUnits_LogMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Log: &LogConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// File is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing file")
	}
}
