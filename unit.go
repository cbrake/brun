package simpleci

import "context"

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
	Check(ctx context.Context) (bool, error)

	// OnTrigger returns the names of units to trigger when condition is met
	OnTrigger() []string
}

// UnitConfig represents the base configuration for all units
type UnitConfig struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Trigger []string `yaml:"trigger,omitempty"`
}
