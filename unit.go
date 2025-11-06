package brun

import "context"

// CheckMode indicates how a trigger unit's Check method is being called
type CheckMode int

const (
	// CheckModePolling indicates Check is called during orchestrator's periodic polling cycle
	CheckModePolling CheckMode = iota

	// CheckModeManual indicates Check is called because another unit triggered this one
	CheckModeManual
)

// String returns a human-readable string for the CheckMode
func (m CheckMode) String() string {
	switch m {
	case CheckModePolling:
		return "polling"
	case CheckModeManual:
		return "manual"
	default:
		return "unknown"
	}
}

// Unit represents a unit of work in the CI system
type Unit interface {
	// Name returns the name of the unit
	Name() string

	// Run executes the unit with the given context
	Run(ctx context.Context) error

	// Type returns the type of unit (e.g., "trigger", "task")
	Type() string
}

// TriggerUnit represents a unit that watches for conditions and triggers other units
type TriggerUnit interface {
	Unit

	// Check returns true if the trigger condition is met
	// mode indicates whether this is a polling check or a manual trigger from another unit
	Check(ctx context.Context, mode CheckMode) (bool, error)

	// OnSuccess returns the names of units to trigger on success
	OnSuccess() []string

	// OnFailure returns the names of units to trigger on failure
	OnFailure() []string

	// Always returns the names of units to trigger regardless of success/failure
	Always() []string
}

// UnitConfig represents the base configuration for all units
type UnitConfig struct {
	Name      string   `yaml:"name"`
	Type      string   `yaml:"type"`
	OnSuccess []string `yaml:"on_success,omitempty"`
	OnFailure []string `yaml:"on_failure,omitempty"`
	Always    []string `yaml:"always,omitempty"`
}
