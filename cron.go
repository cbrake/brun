package brun

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// CronTrigger is a trigger unit that fires based on a cron schedule
type CronTrigger struct {
	name      string
	schedule  string
	state     *State
	parser    cron.Parser
	onSuccess []string
	onFailure []string
	always    []string
}

// CronConfig represents the configuration for a cron trigger
type CronConfig struct {
	UnitConfig `yaml:",inline"`
	Schedule   string `yaml:"schedule"`
}

// NewCronTrigger creates a new cron trigger unit
func NewCronTrigger(name, schedule string, state *State, onSuccess, onFailure, always []string) *CronTrigger {
	return &CronTrigger{
		name:      name,
		schedule:  schedule,
		state:     state,
		parser:    cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the name of the unit
func (c *CronTrigger) Name() string {
	return c.name
}

// Type returns the unit type
func (c *CronTrigger) Type() string {
	return "trigger.cron"
}

// Check returns true if the cron schedule has triggered since the last execution
func (c *CronTrigger) Check(ctx context.Context) (bool, error) {
	// Parse the schedule
	sched, err := c.parser.Parse(c.schedule)
	if err != nil {
		return false, fmt.Errorf("failed to parse cron schedule '%s': %w", c.schedule, err)
	}

	now := time.Now()

	// Get last execution time from state (state is already loaded at startup)
	lastExecStr, ok := c.state.GetString(c.name, "last_execution")
	if !ok {
		// No previous execution, check if we should trigger now
		nextRun := sched.Next(now.Add(-1 * time.Minute))
		if nextRun.Before(now) || nextRun.Equal(now) {
			// Schedule says we should have run, so trigger
			if err := c.state.SetString(c.name, "last_execution", now.Format(time.RFC3339)); err != nil {
				return false, fmt.Errorf("failed to save execution time: %w", err)
			}
			return true, nil
		}
		return false, nil
	}

	// Parse last execution time
	lastExec, err := time.Parse(time.RFC3339, lastExecStr)
	if err != nil {
		// Invalid execution time in state, treat as first run
		if err := c.state.SetString(c.name, "last_execution", now.Format(time.RFC3339)); err != nil {
			return false, fmt.Errorf("failed to save execution time: %w", err)
		}
		return true, nil
	}

	// Check if the schedule indicates we should run
	// Get the next scheduled time after the last execution
	nextRun := sched.Next(lastExec)

	// If the next scheduled run is in the past or now, we should trigger
	if nextRun.Before(now) || nextRun.Equal(now) {
		// Update last execution time
		if err := c.state.SetString(c.name, "last_execution", now.Format(time.RFC3339)); err != nil {
			return false, fmt.Errorf("failed to save execution time: %w", err)
		}
		return true, nil
	}

	return false, nil
}

// OnSuccess returns the list of units to trigger on success
func (c *CronTrigger) OnSuccess() []string {
	return c.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (c *CronTrigger) OnFailure() []string {
	return c.onFailure
}

// Always returns the list of units to trigger regardless of success/failure
func (c *CronTrigger) Always() []string {
	return c.always
}

// Run executes the trigger unit
// Note: Check() has already been called by the orchestrator before Run() is invoked
func (c *CronTrigger) Run(ctx context.Context) error {
	log.Printf("Cron trigger '%s' activated (schedule: %s)", c.name, c.schedule)
	return nil
}
