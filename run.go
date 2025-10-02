package simpleci

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// RunConfig represents the configuration for a Run unit
type RunConfig struct {
	UnitConfig `yaml:",inline"`
	Script     string `yaml:"script"`
	Directory  string `yaml:"directory,omitempty"`
}

// RunUnit executes shell scripts/commands
type RunUnit struct {
	name      string
	script    string
	directory string
	onSuccess []string
	onFailure []string
	always    []string
}

// NewRunUnit creates a new Run unit
func NewRunUnit(name, script, directory string, onSuccess, onFailure, always []string) *RunUnit {
	return &RunUnit{
		name:      name,
		script:    script,
		directory: directory,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the unit name
func (r *RunUnit) Name() string {
	return r.name
}

// Type returns the unit type
func (r *RunUnit) Type() string {
	return "run"
}

// Run executes the shell script
func (r *RunUnit) Run(ctx context.Context) error {
	log.Printf("Running unit '%s'", r.name)

	// Create command to execute script using shell
	cmd := exec.CommandContext(ctx, "sh", "-c", r.script)

	// Set working directory if specified
	if r.directory != "" {
		cmd.Dir = r.directory
		log.Printf("Working directory: %s", r.directory)
	}

	// Set up output to go to stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("script exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute script: %w", err)
	}

	log.Printf("Unit '%s' completed successfully", r.name)
	return nil
}

// OnSuccess returns the list of units to trigger on success
func (r *RunUnit) OnSuccess() []string {
	return r.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (r *RunUnit) OnFailure() []string {
	return r.onFailure
}

// Always returns the list of units to always trigger
func (r *RunUnit) Always() []string {
	return r.always
}
