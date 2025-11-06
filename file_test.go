package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTrigger_Check(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Create some test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "test2.txt")

	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create file trigger with pattern
	pattern := filepath.Join(tempDir, "*.txt")
	trigger := NewFileTrigger(
		"test-file",
		pattern,
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger (new files)
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first check")
	}

	// Second check should not trigger (no changes)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger on second check (no changes)")
	}

	// Modify a file
	if err := os.WriteFile(testFile1, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Third check should trigger (file modified)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger after file modification")
	}

	// Fourth check should not trigger (no new changes)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger after no changes")
	}

	// Add a new file
	testFile3 := filepath.Join(tempDir, "test3.txt")
	if err := os.WriteFile(testFile3, []byte("content3"), 0644); err != nil {
		t.Fatalf("Failed to write new test file: %v", err)
	}

	// Fifth check should trigger (new file added)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger after new file added")
	}

	// Remove a file
	if err := os.Remove(testFile1); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	// Sixth check should trigger (file removed)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger after file removed")
	}
}

func TestFileTrigger_RecursivePattern(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Create nested directories and files
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFile1 := filepath.Join(tempDir, "test.go")
	testFile2 := filepath.Join(subDir, "test.go")

	if err := os.WriteFile(testFile1, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("package sub"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create file trigger with recursive pattern
	pattern := filepath.Join(tempDir, "**/*.go")
	trigger := NewFileTrigger(
		"test-recursive",
		pattern,
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first check")
	}

	// Verify both files were detected
	filesState, err := trigger.getFilesState()
	if err != nil {
		t.Fatalf("Failed to get files state: %v", err)
	}
	if len(filesState) != 2 {
		t.Errorf("Expected 2 files to be detected, got %d", len(filesState))
	}
}

func TestFileTrigger_Run(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create file trigger
	pattern := filepath.Join(tempDir, "*.txt")
	trigger := NewFileTrigger(
		"test-file-run",
		pattern,
		state,
		[]string{"build"},
		[]string{"error"},
		[]string{"always"},
	)

	if trigger.Name() != "test-file-run" {
		t.Errorf("Expected name 'test-file-run', got '%s'", trigger.Name())
	}

	if trigger.Type() != "trigger.file" {
		t.Errorf("Expected type 'trigger.file', got '%s'", trigger.Type())
	}

	ctx := context.Background()
	if err := trigger.Run(ctx); err != nil {
		t.Errorf("Run failed: %v", err)
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
	if len(always) != 1 || always[0] != "always" {
		t.Errorf("Expected always [always], got %v", always)
	}
}

func TestLoadConfig_WithFileUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - file:
      name: watch-files
      pattern: "**/*.go"
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
	if unit.Name() != "watch-files" {
		t.Errorf("Expected name 'watch-files', got '%s'", unit.Name())
	}

	if unit.Type() != "trigger.file" {
		t.Errorf("Expected type 'trigger.file', got '%s'", unit.Type())
	}

	fileTrigger, ok := unit.(*FileTrigger)
	if !ok {
		t.Fatal("Unit is not a FileTrigger")
	}

	if fileTrigger.pattern != "**/*.go" {
		t.Errorf("Expected pattern '**/*.go', got '%s'", fileTrigger.pattern)
	}

	if len(fileTrigger.onSuccess) != 1 || fileTrigger.onSuccess[0] != "build" {
		t.Errorf("Expected on_success [build], got %v", fileTrigger.onSuccess)
	}
}

func TestCreateUnits_FileMissingPattern(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				File: &FileConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// Pattern is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing pattern")
	}
}

func TestFileTrigger_InvalidPattern(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	state := NewState(stateFile)
	trigger := NewFileTrigger(
		"test-invalid",
		"[",
		state,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	_, err := trigger.Check(ctx, CheckModePolling)
	if err == nil {
		t.Error("Expected error for invalid pattern")
	}
}

func TestFileTrigger_IgnoresDirectories(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Create a directory that matches pattern
	dirPath := filepath.Join(tempDir, "test.txt")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create a file
	testFile := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create file trigger
	pattern := filepath.Join(tempDir, "*.txt")
	trigger := NewFileTrigger(
		"test-dirs",
		pattern,
		state,
		nil,
		nil,
		nil,
	)

	// Get files state
	filesState, err := trigger.getFilesState()
	if err != nil {
		t.Fatalf("Failed to get files state: %v", err)
	}

	// Should only have 1 file (directory should be ignored)
	if len(filesState) != 1 {
		t.Errorf("Expected 1 file, got %d (directories should be ignored)", len(filesState))
	}

	// Verify it's the file, not the directory
	if _, ok := filesState[testFile]; !ok {
		t.Error("Expected file to be in files state")
	}
	if _, ok := filesState[dirPath]; ok {
		t.Error("Directory should not be in files state")
	}
}
