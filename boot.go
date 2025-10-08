package brun

import (
	"context"
	"fmt"
	"time"
)

// BootTrigger is a trigger unit that fires on the first run after system boot
type BootTrigger struct {
	name      string
	state     *State
	onSuccess []string
	onFailure []string
	always    []string
}

// BootConfig represents the configuration for a boot trigger
type BootConfig struct {
	UnitConfig `yaml:",inline"`
}

// NewBootTrigger creates a new boot trigger unit
func NewBootTrigger(name string, state *State, onSuccess, onFailure, always []string) *BootTrigger {
	return &BootTrigger{
		name:      name,
		state:     state,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the name of the unit
func (s *BootTrigger) Name() string {
	return s.name
}

// Type returns the unit type
func (s *BootTrigger) Type() string {
	return "trigger.boot"
}

// Check returns true if this is the first run since system boot
func (s *BootTrigger) Check(ctx context.Context) (bool, error) {
	// Get current boot time
	detector := NewBootDetector("")
	currentBootTime, err := detector.GetBootTime()
	if err != nil {
		return false, fmt.Errorf("failed to get boot time: %w", err)
	}

	// Get last boot time from state (state is already loaded at startup)
	lastBootTimeStr, ok := s.state.GetString(s.name, "last_boot_time")
	if !ok {
		// No previous boot time, this is the first run
		if err := s.state.SetString(s.name, "last_boot_time", currentBootTime.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return false, fmt.Errorf("failed to save boot time: %w", err)
		}
		if err := s.state.Set(s.name, "boot_count", 1); err != nil {
			return false, fmt.Errorf("failed to save boot count: %w", err)
		}
		return true, nil
	}

	// Parse last boot time
	lastBootTime, err := parseBootTime(lastBootTimeStr)
	if err != nil {
		// Invalid boot time in state, treat as first run
		if err := s.state.SetString(s.name, "last_boot_time", currentBootTime.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return false, fmt.Errorf("failed to save boot time: %w", err)
		}
		if err := s.state.Set(s.name, "boot_count", 1); err != nil {
			return false, fmt.Errorf("failed to save boot count: %w", err)
		}
		return true, nil
	}

	// Check if boot time has changed (with 10 second tolerance)
	diff := currentBootTime.Sub(lastBootTime)
	if diff < 0 {
		diff = -diff
	}

	isFirstRun := diff > 10*time.Second
	if isFirstRun {
		// Get current boot count
		bootCount := 1
		if countVal, ok := s.state.Get(s.name, "boot_count"); ok {
			if count, ok := countVal.(int); ok {
				bootCount = count + 1
			}
		}

		// Update state with new boot time and incremented boot count
		if err := s.state.SetString(s.name, "last_boot_time", currentBootTime.Format("2006-01-02T15:04:05Z07:00")); err != nil {
			return false, fmt.Errorf("failed to save boot time: %w", err)
		}
		if err := s.state.Set(s.name, "boot_count", bootCount); err != nil {
			return false, fmt.Errorf("failed to save boot count: %w", err)
		}
	}

	return isFirstRun, nil
}

// OnSuccess returns the list of units to trigger on success
func (s *BootTrigger) OnSuccess() []string {
	return s.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (s *BootTrigger) OnFailure() []string {
	return s.onFailure
}

// Always returns the list of units to trigger regardless of success/failure
func (s *BootTrigger) Always() []string {
	return s.always
}

// Run executes the trigger unit
// Note: Check() has already been called by the orchestrator before Run() is invoked
func (s *BootTrigger) Run(ctx context.Context) error {
	// Get boot count from state
	bootCount := 1
	if countVal, ok := s.state.Get(s.name, "boot_count"); ok {
		if count, ok := countVal.(int); ok {
			bootCount = count
		}
	}

	fmt.Printf("Boot trigger '%s' activated (boot count: %d)\n", s.name, bootCount)
	return nil
}

// parseBootTime parses a boot time string in RFC3339 format
func parseBootTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z07:00", s)
}
