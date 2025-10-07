package brun

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

// RunConfig represents the configuration for a Run unit
type RunConfig struct {
	UnitConfig `yaml:",inline"`
	Script     string `yaml:"script"`
	Directory  string `yaml:"directory,omitempty"`
	Timeout    string `yaml:"timeout,omitempty"`
	Shell      string `yaml:"shell,omitempty"`
	UsePTY     bool   `yaml:"use_pty,omitempty"`
}

// RunUnit executes shell scripts/commands
type RunUnit struct {
	name      string
	script    string
	directory string
	timeout   time.Duration
	shell     string
	usePTY    bool
	onSuccess []string
	onFailure []string
	always    []string
}

// NewRunUnit creates a new Run unit
func NewRunUnit(name, script, directory string, timeout time.Duration, shell string, usePTY bool, onSuccess, onFailure, always []string) *RunUnit {
	// Default to 'sh' if no shell is specified
	if shell == "" {
		shell = "sh"
	}
	return &RunUnit{
		name:      name,
		script:    script,
		directory: directory,
		timeout:   timeout,
		shell:     shell,
		usePTY:    usePTY,
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

	// Apply timeout if configured
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
		log.Printf("Timeout set to %s", r.timeout)
	}

	// Create command to execute script using configured shell
	var cmd *exec.Cmd
	if r.usePTY {
		// Wrap command with 'script' to provide a pseudo-TTY
		// Build the command as: script -q -e -c "bash -c 'script contents'" /dev/null
		// We need to pass each argument separately to avoid quote interpretation issues
		scriptPath, _ := exec.LookPath("script")
		cmd = &exec.Cmd{
			Path: scriptPath,
			Args: []string{"script", "-q", "-e", "-c", r.shell, "-c", r.script, "/dev/null"},
		}
		if ctx != nil {
			cmd = exec.CommandContext(ctx, scriptPath, "-q", "-e", "-c", r.shell, "-c", r.script, "/dev/null")
		}
	} else {
		cmd = exec.CommandContext(ctx, r.shell, "-c", r.script)
	}

	// Set working directory if specified
	if r.directory != "" {
		cmd.Dir = r.directory
		log.Printf("Working directory: %s", r.directory)
	}

	// Set up output to go to stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Inherit environment and set TERM to ensure tools expecting shell environment work
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Run the command
	if err := cmd.Run(); err != nil {
		// Check if error is due to context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("task timed out after %s", r.timeout)
		}
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
