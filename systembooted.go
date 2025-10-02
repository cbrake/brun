package simpleci

import (
	"context"
	"fmt"
)

// SystemBootedTrigger is a trigger unit that fires on the first run after system boot
type SystemBootedTrigger struct {
	name         string
	detector     *BootDetector
	triggerUnits []string
}

// SystemBootedConfig represents the configuration for a system booted trigger
type SystemBootedConfig struct {
	UnitConfig `yaml:",inline"`
	StateFile  string `yaml:"state_file,omitempty"`
}

// NewSystemBootedTrigger creates a new system booted trigger unit
func NewSystemBootedTrigger(name string, stateFile string, triggerUnits []string) *SystemBootedTrigger {
	if stateFile == "" {
		stateFile = "/var/lib/simpleci/systembooted.state"
	}

	return &SystemBootedTrigger{
		name:         name,
		detector:     NewBootDetector(stateFile),
		triggerUnits: triggerUnits,
	}
}

// Name returns the name of the unit
func (s *SystemBootedTrigger) Name() string {
	return s.name
}

// Type returns the unit type
func (s *SystemBootedTrigger) Type() string {
	return "trigger.systembooted"
}

// Check returns true if this is the first run since system boot
func (s *SystemBootedTrigger) Check(ctx context.Context) (bool, error) {
	isFirstRun, err := s.detector.IsFirstRunSinceBoot()
	if err != nil {
		return false, fmt.Errorf("failed to check boot status: %w", err)
	}
	return isFirstRun, nil
}

// OnTrigger returns the list of units to trigger
func (s *SystemBootedTrigger) OnTrigger() []string {
	return s.triggerUnits
}

// Run executes the trigger unit
func (s *SystemBootedTrigger) Run(ctx context.Context) error {
	triggered, err := s.Check(ctx)
	if err != nil {
		return err
	}

	if triggered {
		fmt.Printf("System booted trigger '%s' activated\n", s.name)
		// In a full implementation, this would trigger the downstream units
		// For now, we just report the trigger
	}

	return nil
}
