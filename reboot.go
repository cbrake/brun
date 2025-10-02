package simpleci

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// RebootUnit is a unit that logs and reboots the system
type RebootUnit struct {
	name      string
	delay     int // delay in seconds before reboot
	onSuccess []string
	onFailure []string
	always    []string
}

// RebootConfig represents the configuration for a reboot unit
type RebootConfig struct {
	UnitConfig `yaml:",inline"`
	Delay      int `yaml:"delay,omitempty"` // delay in seconds before reboot
}

// NewRebootUnit creates a new reboot unit
func NewRebootUnit(name string, delay int, onSuccess, onFailure, always []string) *RebootUnit {
	if delay <= 0 {
		delay = 0 // immediate reboot
	}

	return &RebootUnit{
		name:      name,
		delay:     delay,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the name of the unit
func (r *RebootUnit) Name() string {
	return r.name
}

// Type returns the unit type
func (r *RebootUnit) Type() string {
	return "reboot"
}

// Run executes the reboot unit
func (r *RebootUnit) Run(ctx context.Context) error {
	fmt.Printf("Reboot unit '%s' executing\n", r.name)

	if r.delay > 0 {
		fmt.Printf("Rebooting in %d seconds...\n", r.delay)
		time.Sleep(time.Duration(r.delay) * time.Second)
	} else {
		fmt.Println("Rebooting now...")
	}

	// Execute reboot command
	cmd := exec.Command("reboot")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute reboot: %w", err)
	}

	return nil
}

// OnSuccess returns the list of units to trigger on success
func (r *RebootUnit) OnSuccess() []string {
	return r.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (r *RebootUnit) OnFailure() []string {
	return r.onFailure
}

// Always returns the list of units to trigger regardless of success/failure
func (r *RebootUnit) Always() []string {
	return r.always
}
