package simpleci

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	systemServicePath = "/etc/systemd/system/simpleci.service"
	userServiceDir    = ".config/systemd/user"
	userServiceName   = "simpleci.service"
)

// Install installs simpleci as a systemd service
// If run as root, installs system-wide service
// Otherwise, installs user service
func Install() error {
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
		return installSystemService(execPath)
	}
	return installUserService(execPath)
}

// installSystemService installs a system-wide systemd service
func installSystemService(execPath string) error {
	fmt.Println("Installing system-wide systemd service...")

	serviceContent := generateSystemServiceFile(execPath)

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
	if err := exec.Command("systemctl", "enable", "simpleci.service").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	fmt.Println("Service enabled. Start it with: systemctl start simpleci.service")
	return nil
}

// installUserService installs a user systemd service
func installUserService(execPath string) error {
	fmt.Println("Installing user systemd service...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	serviceDir := filepath.Join(homeDir, userServiceDir)
	servicePath := filepath.Join(serviceDir, userServiceName)

	// Create service directory if it doesn't exist
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	serviceContent := generateUserServiceFile(execPath)

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

	fmt.Println("Service enabled. Start it with: systemctl --user start simpleci.service")
	return nil
}

// generateSystemServiceFile generates the systemd service file content for system service
func generateSystemServiceFile(execPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Simple CI - Continuous Integration for Hardware Testing
After=network.target

[Service]
Type=oneshot
ExecStart=%s /etc/simpleci/config.yaml
StandardOutput=journal
StandardError=journal
Restart=no

[Install]
WantedBy=multi-user.target
`, execPath)
}

// generateUserServiceFile generates the systemd service file content for user service
func generateUserServiceFile(execPath string) string {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "simpleci", "config.yaml")

	return fmt.Sprintf(`[Unit]
Description=Simple CI - Continuous Integration for Hardware Testing
After=network.target

[Service]
Type=oneshot
ExecStart=%s %s
StandardOutput=journal
StandardError=journal
Restart=no

[Install]
WantedBy=default.target
`, execPath, configPath)
}
