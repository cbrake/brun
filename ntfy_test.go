package brun

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNtfyUnit_Basic(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"Build Alert",
		"high",
		"warning,skull",
		true,
		50,
		[]string{"next-unit"},
		[]string{"error-unit"},
		[]string{"always-unit"},
	)

	if unit.Name() != "test-ntfy" {
		t.Errorf("Expected name 'test-ntfy', got '%s'", unit.Name())
	}

	if unit.Type() != "ntfy" {
		t.Errorf("Expected type 'ntfy', got '%s'", unit.Type())
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

func TestNtfyUnit_SetOutput(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	testOutput := "Build output here\nLine 2"
	unit.SetOutput(testOutput)

	if unit.output != testOutput {
		t.Errorf("Expected output '%s', got '%s'", testOutput, unit.output)
	}
}

func TestNtfyUnit_SetTriggeringUnit(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")

	if unit.triggeringUnit != "build-unit" {
		t.Errorf("Expected triggering unit 'build-unit', got '%s'", unit.triggeringUnit)
	}
}

func TestNtfyUnit_SetTriggerError(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	testErr := errors.New("build failed")
	unit.SetTriggerError(testErr)

	if unit.triggerError != testErr {
		t.Errorf("Expected trigger error '%v', got '%v'", testErr, unit.triggerError)
	}
}

func TestNtfyUnit_BuildBody(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetOutput("Line 1\nLine 2\nLine 3")

	body := unit.buildBody()

	if !strings.Contains(body, "Triggered by: build-unit") {
		t.Error("Body missing triggering unit")
	}

	if !strings.Contains(body, "Timestamp:") {
		t.Error("Body missing timestamp")
	}

	if !strings.Contains(body, "Output:") {
		t.Error("Body missing output section")
	}

	if !strings.Contains(body, "Line 1\nLine 2\nLine 3") {
		t.Error("Body missing output content")
	}
}

func TestNtfyUnit_BuildBody_WithError(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetTriggerError(errors.New("exit status 1"))

	body := unit.buildBody()

	if !strings.Contains(body, "Error: exit status 1") {
		t.Error("Body missing error")
	}
}

func TestNtfyUnit_BuildBody_LimitLines(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		2, // Limit to 2 lines
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetOutput("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	body := unit.buildBody()

	// Should only contain last 2 lines
	if !strings.Contains(body, "Line 4\nLine 5") {
		t.Error("Body should contain last 2 lines")
	}

	if strings.Contains(body, "Line 1") {
		t.Error("Body should not contain Line 1")
	}

	if !strings.Contains(body, "(last 2 of 5 lines)") {
		t.Error("Body missing line limit notice")
	}
}

func TestNtfyUnit_BuildBody_NoOutput(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	// No output set

	body := unit.buildBody()

	if !strings.Contains(body, "(No output captured)") {
		t.Error("Body should indicate no output captured")
	}
}

func TestNtfyUnit_BuildBody_OutputDisabled(t *testing.T) {
	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		"https://ntfy.sh",
		"",
		"",
		"",
		false, // includeOutput = false
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetOutput("Some output")

	body := unit.buildBody()

	if !strings.Contains(body, "(Output not included)") {
		t.Error("Body should indicate output not included")
	}

	if strings.Contains(body, "Some output") {
		t.Error("Body should not contain output when disabled")
	}
}

func TestNtfyUnit_Run_Success(t *testing.T) {
	// Create a mock server
	var receivedTitle, receivedPriority, receivedTags, receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTitle = r.Header.Get("Title")
		receivedPriority = r.Header.Get("Priority")
		receivedTags = r.Header.Get("Tags")
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		server.URL,
		"Build Alert",
		"high",
		"warning",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetOutput("Build succeeded!")

	err := unit.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check title format: <prefix>: <unit>:<status>
	if receivedTitle != "Build Alert: build-unit:success" {
		t.Errorf("Expected title 'Build Alert: build-unit:success', got '%s'", receivedTitle)
	}

	if receivedPriority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", receivedPriority)
	}

	if receivedTags != "warning" {
		t.Errorf("Expected tags 'warning', got '%s'", receivedTags)
	}

	if !strings.Contains(receivedBody, "Build succeeded!") {
		t.Error("Body should contain output")
	}
}

func TestNtfyUnit_Run_Failure(t *testing.T) {
	var receivedTitle string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTitle = r.Header.Get("Title")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		server.URL,
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	unit.SetTriggeringUnit("build-unit")
	unit.SetTriggerError(errors.New("exit status 1"))

	err := unit.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check title format without prefix: <unit>:<status>
	if receivedTitle != "build-unit:fail" {
		t.Errorf("Expected title 'build-unit:fail', got '%s'", receivedTitle)
	}
}

func TestNtfyUnit_Run_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	unit := NewNtfyUnit(
		"test-ntfy",
		"my-topic",
		server.URL,
		"",
		"",
		"",
		true,
		0,
		nil,
		nil,
		nil,
	)

	err := unit.Run(context.Background())
	if err == nil {
		t.Error("Expected error for server error response")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Error should mention status code, got: %v", err)
	}
}

func TestLoadConfig_WithNtfyUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - ntfy:
      name: notify-failure
      topic: my-build-alerts
      server: https://custom.ntfy.sh
      title_prefix: "Build Alert"
      priority: high
      tags: warning,skull
      include_output: false
      limit_lines: 50
      on_success:
        - next-step
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
	if unit.Name() != "notify-failure" {
		t.Errorf("Expected name 'notify-failure', got '%s'", unit.Name())
	}

	if unit.Type() != "ntfy" {
		t.Errorf("Expected type 'ntfy', got '%s'", unit.Type())
	}

	ntfyUnit, ok := unit.(*NtfyUnit)
	if !ok {
		t.Fatal("Unit is not an NtfyUnit")
	}

	if ntfyUnit.topic != "my-build-alerts" {
		t.Errorf("Expected topic 'my-build-alerts', got '%s'", ntfyUnit.topic)
	}

	if ntfyUnit.server != "https://custom.ntfy.sh" {
		t.Errorf("Expected server 'https://custom.ntfy.sh', got '%s'", ntfyUnit.server)
	}

	if ntfyUnit.titlePrefix != "Build Alert" {
		t.Errorf("Expected title_prefix 'Build Alert', got '%s'", ntfyUnit.titlePrefix)
	}

	if ntfyUnit.priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", ntfyUnit.priority)
	}

	if ntfyUnit.tags != "warning,skull" {
		t.Errorf("Expected tags 'warning,skull', got '%s'", ntfyUnit.tags)
	}

	if ntfyUnit.includeOutput {
		t.Error("Expected include_output to be false")
	}

	if ntfyUnit.limitLines != 50 {
		t.Errorf("Expected limit_lines 50, got %d", ntfyUnit.limitLines)
	}

	if len(ntfyUnit.onSuccess) != 1 || ntfyUnit.onSuccess[0] != "next-step" {
		t.Errorf("Expected on_success [next-step], got %v", ntfyUnit.onSuccess)
	}
}

func TestLoadConfig_WithNtfyDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - ntfy:
      name: notify
      topic: my-alerts
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

	ntfyUnit := units[0].(*NtfyUnit)

	// Check defaults
	if ntfyUnit.server != "https://ntfy.sh" {
		t.Errorf("Expected default server 'https://ntfy.sh', got '%s'", ntfyUnit.server)
	}

	if !ntfyUnit.includeOutput {
		t.Error("Expected default include_output to be true")
	}
}

func TestCreateUnits_NtfyMissingTopic(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Ntfy: &NtfyConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// Topic is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing topic")
	}
}

func TestCreateUnits_NtfyMissingName(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Ntfy: &NtfyConfig{
					Topic: "my-topic",
					// Name is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing name")
	}
}
