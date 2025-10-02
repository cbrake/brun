package metalci

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// BootDetector detects system boot time and tracks whether this is the first run since boot
type BootDetector struct {
	stateFile string
}

// NewBootDetector creates a new boot detector with the given state file path
func NewBootDetector(stateFile string) *BootDetector {
	return &BootDetector{
		stateFile: stateFile,
	}
}

// GetBootTime returns the system boot time by reading /proc/uptime
func (bd *BootDetector) GetBootTime() (time.Time, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read /proc/uptime: %w", err)
	}

	// /proc/uptime contains two numbers: system uptime and idle time in seconds
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return time.Time{}, fmt.Errorf("invalid /proc/uptime format")
	}

	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse uptime: %w", err)
	}

	bootTime := time.Now().Add(-time.Duration(uptimeSeconds * float64(time.Second)))
	return bootTime, nil
}

// IsFirstRunSinceBoot checks if this is the first run since system boot
// It compares the current boot time with the stored boot time from the last run
func (bd *BootDetector) IsFirstRunSinceBoot() (bool, error) {
	currentBootTime, err := bd.GetBootTime()
	if err != nil {
		return false, err
	}

	// Read the last recorded boot time
	data, err := os.ReadFile(bd.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist, this is the first run ever or since boot
			return true, bd.recordBootTime(currentBootTime)
		}
		return false, fmt.Errorf("failed to read state file: %w", err)
	}

	lastBootTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		// Invalid state file, treat as first run
		return true, bd.recordBootTime(currentBootTime)
	}

	// If current boot time is significantly different from last boot time, system has rebooted
	// We use a 10 second tolerance to account for minor variations in calculation
	diff := currentBootTime.Sub(lastBootTime)
	if diff < 0 {
		diff = -diff
	}

	isFirstRun := diff > 10*time.Second

	if isFirstRun {
		// Update state file with new boot time
		if err := bd.recordBootTime(currentBootTime); err != nil {
			return false, err
		}
	}

	return isFirstRun, nil
}

// recordBootTime writes the boot time to the state file
func (bd *BootDetector) recordBootTime(bootTime time.Time) error {
	// Ensure directory exists
	dir := bd.stateFile[:strings.LastIndex(bd.stateFile, "/")]
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}
	}

	data := []byte(bootTime.Format(time.RFC3339))
	if err := os.WriteFile(bd.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
