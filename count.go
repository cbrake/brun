package brun

import (
	"context"
	"fmt"
	"log"
)

// CountConfig represents the configuration for a Count unit
type CountConfig struct {
	UnitConfig `yaml:",inline"`
}

// CountUnit tracks how many times it has been triggered by each unit
type CountUnit struct {
	name           string
	state          *State
	triggeringUnit string // Name of the unit that triggered this count
	onSuccess      []string
	onFailure      []string
	always         []string
}

// NewCountUnit creates a new Count unit
func NewCountUnit(name string, state *State, onSuccess, onFailure, always []string) *CountUnit {
	return &CountUnit{
		name:      name,
		state:     state,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the unit name
func (c *CountUnit) Name() string {
	return c.name
}

// Type returns the unit type
func (c *CountUnit) Type() string {
	return "count"
}

// SetTriggeringUnit sets the name of the unit that triggered this count
func (c *CountUnit) SetTriggeringUnit(unitName string) {
	c.triggeringUnit = unitName
}

// Run executes the count unit
func (c *CountUnit) Run(ctx context.Context) error {
	log.Printf("Running count unit '%s'", c.name)

	// Determine the triggering unit name (state is already loaded at startup)
	unitName := c.triggeringUnit
	if unitName == "" {
		unitName = "unknown"
	}

	// Get current count for this triggering unit
	currentCount := 0
	if val, ok := c.state.Get(c.name, unitName); ok {
		if intVal, ok := val.(int); ok {
			currentCount = intVal
		}
	}

	// Increment count
	newCount := currentCount + 1

	// Save to state
	if err := c.state.Set(c.name, unitName, newCount); err != nil {
		return fmt.Errorf("failed to save count: %w", err)
	}

	log.Printf("Count unit '%s': unit '%s' has triggered %d time(s)", c.name, unitName, newCount)
	return nil
}

// OnSuccess returns the list of units to trigger on success
func (c *CountUnit) OnSuccess() []string {
	return c.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (c *CountUnit) OnFailure() []string {
	return c.onFailure
}

// Always returns the list of units to always trigger
func (c *CountUnit) Always() []string {
	return c.always
}
