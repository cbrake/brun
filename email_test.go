package brun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmailUnit_Basic(t *testing.T) {
	unit := NewEmailUnit(
		"test-email",
		[]string{"user@example.com"},
		"sender@example.com",
		"Test Prefix",
		"smtp.example.com",
		587,
		"user",
		"pass",
		true,
		true,
		0,
		[]string{"next-unit"},
		[]string{"error-unit"},
		[]string{"always-unit"},
	)

	if unit.Name() != "test-email" {
		t.Errorf("Expected name 'test-email', got '%s'", unit.Name())
	}

	if unit.Type() != "email" {
		t.Errorf("Expected type 'email', got '%s'", unit.Type())
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

func TestEmailUnit_SetOutput(t *testing.T) {
	unit := NewEmailUnit(
		"test-email",
		[]string{"user@example.com"},
		"sender@example.com",
		"Test Subject",
		"smtp.example.com",
		587,
		"",
		"",
		false,
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

func TestEmailUnit_SetTriggeringUnit(t *testing.T) {
	unit := NewEmailUnit(
		"test-email",
		[]string{"user@example.com"},
		"sender@example.com",
		"",
		"smtp.example.com",
		587,
		"",
		"",
		false,
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

func TestEmailUnit_BuildMessage(t *testing.T) {
	unit := NewEmailUnit(
		"test-email",
		[]string{"user1@example.com", "user2@example.com"},
		"sender@example.com",
		"Alert",
		"smtp.example.com",
		587,
		"",
		"",
		false,
		true,
		0,
		nil,
		nil,
		nil,
	)

	// Set triggering unit to test subject building
	unit.SetTriggeringUnit("build-unit")

	message := unit.buildMessage("Alert: build-unit:success", "Test Body")

	if !strings.Contains(message, "From: sender@example.com") {
		t.Error("Message missing From header")
	}

	if !strings.Contains(message, "To: user1@example.com, user2@example.com") {
		t.Error("Message missing To header")
	}

	if !strings.Contains(message, "Subject: Alert: build-unit:success") {
		t.Error("Message missing Subject header")
	}

	if !strings.Contains(message, "Test Body") {
		t.Error("Message missing body")
	}

	if !strings.Contains(message, "MIME-Version: 1.0") {
		t.Error("Message missing MIME-Version header")
	}
}

func TestLoadConfig_WithEmailUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - email:
      name: notify-failure
      to:
        - admin@example.com
        - alerts@example.com
      from: brun@example.com
      subject_prefix: "Build Alert"
      smtp_host: smtp.example.com
      smtp_port: 587
      smtp_user: brun@example.com
      smtp_password: secret
      smtp_use_tls: true
      include_output: false
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

	if unit.Type() != "email" {
		t.Errorf("Expected type 'email', got '%s'", unit.Type())
	}

	emailUnit, ok := unit.(*EmailUnit)
	if !ok {
		t.Fatal("Unit is not an EmailUnit")
	}

	if len(emailUnit.to) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(emailUnit.to))
	}

	if emailUnit.from != "brun@example.com" {
		t.Errorf("Expected from 'brun@example.com', got '%s'", emailUnit.from)
	}

	if emailUnit.subjectPrefix != "Build Alert" {
		t.Errorf("Expected subject_prefix 'Build Alert', got '%s'", emailUnit.subjectPrefix)
	}

	if emailUnit.smtpHost != "smtp.example.com" {
		t.Errorf("Expected smtp_host 'smtp.example.com', got '%s'", emailUnit.smtpHost)
	}

	if emailUnit.smtpPort != 587 {
		t.Errorf("Expected smtp_port 587, got %d", emailUnit.smtpPort)
	}

	if emailUnit.smtpUser != "brun@example.com" {
		t.Errorf("Expected smtp_user 'brun@example.com', got '%s'", emailUnit.smtpUser)
	}

	if emailUnit.smtpPassword != "secret" {
		t.Errorf("Expected smtp_password 'secret', got '%s'", emailUnit.smtpPassword)
	}

	if !emailUnit.smtpUseTLS {
		t.Error("Expected smtp_use_tls to be true")
	}

	if emailUnit.includeOutput {
		t.Error("Expected include_output to be false")
	}

	if len(emailUnit.onSuccess) != 1 || emailUnit.onSuccess[0] != "next-step" {
		t.Errorf("Expected on_success [next-step], got %v", emailUnit.onSuccess)
	}
}

func TestLoadConfig_WithEmailDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - email:
      name: notify
      to:
        - admin@example.com
      from: brun@example.com
      smtp_host: smtp.example.com
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

	emailUnit := units[0].(*EmailUnit)

	// Check defaults
	if emailUnit.smtpPort != 587 {
		t.Errorf("Expected default smtp_port 587, got %d", emailUnit.smtpPort)
	}

	if !emailUnit.smtpUseTLS {
		t.Error("Expected default smtp_use_tls to be true")
	}

	if !emailUnit.includeOutput {
		t.Error("Expected default include_output to be true")
	}
}

func TestCreateUnits_EmailMissingTo(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Email: &EmailConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					From:     "sender@example.com",
					SMTPHost: "smtp.example.com",
					// To is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing to")
	}
}

func TestCreateUnits_EmailMissingFrom(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Email: &EmailConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					To:       []string{"user@example.com"},
					SMTPHost: "smtp.example.com",
					// From is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing from")
	}
}

func TestCreateUnits_EmailMissingSMTPHost(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Email: &EmailConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					To:   []string{"user@example.com"},
					From: "sender@example.com",
					// SMTPHost is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing smtp_host")
	}
}
