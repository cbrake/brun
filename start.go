package brun

import (
	"context"
	"log"
)

// StartConfig represents the configuration for a Start trigger
type StartConfig struct {
	UnitConfig `yaml:",inline"`
}

// StartTrigger is a trigger that always fires when brun starts
type StartTrigger struct {
	name      string
	onSuccess []string
	onFailure []string
	always    []string
}

// NewStartTrigger creates a new Start trigger
func NewStartTrigger(name string, onSuccess, onFailure, always []string) *StartTrigger {
	return &StartTrigger{
		name:      name,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the trigger name
func (s *StartTrigger) Name() string {
	return s.name
}

// Type returns the trigger type
func (s *StartTrigger) Type() string {
	return "trigger.start"
}

// Check always returns true since this trigger fires on every run
func (s *StartTrigger) Check(ctx context.Context, mode CheckMode) (bool, error) {
	// Start always triggers, regardless of mode
	return true, nil
}

// Run executes when the trigger fires
func (s *StartTrigger) Run(ctx context.Context) error {
	log.Printf("Start trigger '%s' activated", s.name)
	return nil
}

// OnSuccess returns the list of units to trigger on success
func (s *StartTrigger) OnSuccess() []string {
	return s.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (s *StartTrigger) OnFailure() []string {
	return s.onFailure
}

// Always returns the list of units to always trigger
func (s *StartTrigger) Always() []string {
	return s.always
}
