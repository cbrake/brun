package brun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	systemServicePath = "/etc/systemd/system/brun.service"
	userServiceDir    = ".config/systemd/user"
	userServiceName   = "brun.service"
)

// Install installs brun as a systemd service
// If run as root, installs system-wide service
// Otherwise, installs user service
// daemonMode determines whether the service runs in daemon mode (continuous) or oneshot mode
func Install(daemonMode bool) error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get actual path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Check if running as root
	isRoot := os.Geteuid() == 0

	if isRoot {
		return installSystemService(execPath, daemonMode)
	}
	return installUserService(execPath, daemonMode)
}

// installSystemService installs a system-wide systemd service
func installSystemService(execPath string, daemonMode bool) error {
	fmt.Println("Installing system-wide systemd service...")

	configPath := "/etc/brun/config.yaml"

	// Create default config if it doesn't exist
	if err := createDefaultConfigIfNeeded(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	serviceContent := generateSystemServiceFile(execPath, daemonMode)

	// Write service file
	if err := os.WriteFile(systemServicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	fmt.Printf("Service file written to %s\n", systemServicePath)

	// Reload systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	if err := exec.Command("systemctl", "enable", "brun.service").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("Service enabled. Start it with: systemctl start brun.service")
	return nil
}

// installUserService installs a user systemd service
func installUserService(execPath string, daemonMode bool) error {
	fmt.Println("Installing user systemd service...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "brun", "config.yaml")

	// Create default config if it doesn't exist
	if err := createDefaultConfigIfNeeded(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	serviceDir := filepath.Join(homeDir, userServiceDir)
	servicePath := filepath.Join(serviceDir, userServiceName)

	// Create service directory if it doesn't exist
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	serviceContent := generateUserServiceFile(execPath, daemonMode)

	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	fmt.Printf("Service file written to %s\n", servicePath)

	// Reload user systemd
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable user service
	if err := exec.Command("systemctl", "--user", "enable", userServiceName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("Service enabled. Start it with: systemctl --user start brun.service")
	return nil
}

// generateSystemServiceFile generates the systemd service file content for system service
func generateSystemServiceFile(execPath string, daemonMode bool) string {
	serviceType := "oneshot"
	execCommand := fmt.Sprintf("%s run /etc/brun/config.yaml", execPath)
	restart := "no"

	if daemonMode {
		serviceType = "simple"
		execCommand = fmt.Sprintf("%s run /etc/brun/config.yaml -daemon", execPath)
		restart = "always"
	}

	return fmt.Sprintf(`[Unit]
Description=BRun - Bare-OS Runner
After=network.target

[Service]
Type=%s
ExecStart=%s
StandardOutput=journal
StandardError=journal
Restart=%s

[Install]
WantedBy=multi-user.target
`, serviceType, execCommand, restart)
}

// generateUserServiceFile generates the systemd service file content for user service
func generateUserServiceFile(execPath string, daemonMode bool) string {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "brun", "config.yaml")

	serviceType := "oneshot"
	execCommand := fmt.Sprintf("%s run %s", execPath, configPath)
	restart := "no"

	if daemonMode {
		serviceType = "simple"
		execCommand = fmt.Sprintf("%s run %s -daemon", execPath, configPath)
		restart = "always"
	}

	return fmt.Sprintf(`[Unit]
Description=BRun - Bare-OS Runner
After=network.target

[Service]
Type=%s
ExecStart=%s
StandardOutput=journal
StandardError=journal
Restart=%s

[Install]
WantedBy=default.target
`, serviceType, execCommand, restart)
}

// createDefaultConfigIfNeeded creates a default config file if one doesn't exist
func createDefaultConfigIfNeeded(configPath string) error {
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file already exists at %s\n", configPath)
		return nil
	}

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Default config content - use different state location for root vs user
	stateLocation := "/var/lib/brun/state.yaml"
	if configPath != "/etc/brun/config.yaml" {
		// User install
		homeDir, _ := os.UserHomeDir()
		stateLocation = filepath.Join(homeDir, ".config", "brun", "state.yaml")
	}

	defaultConfig := fmt.Sprintf(`# BRun Configuration File
# See https://github.com/cbrake/brun for documentation

config:
  state_location: %s

units:
  - boot:
      name: boot-trigger
      on_success:
        - build-unit
        - test-unit

  # Add your units here
  # - reboot:
  #     name: reboot-system
  #     delay: 5
`, stateLocation)

	// Write default config
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created default config file at %s\n", configPath)
	return nil
}
