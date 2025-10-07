package brun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// UnitResult represents the result of a unit execution
type UnitResult struct {
	Unit   Unit
	Error  error
	Output string // Captured stdout/stderr
}

// Orchestrator manages unit execution and triggering
type Orchestrator struct {
	units       []Unit
	unitsByName map[string]Unit
	results     map[string]*UnitResult
}

// NewOrchestrator creates a new orchestrator with the given units
func NewOrchestrator(units []Unit) *Orchestrator {
	unitsByName := make(map[string]Unit)
	for _, unit := range units {
		unitsByName[unit.Name()] = unit
	}

	return &Orchestrator{
		units:       units,
		unitsByName: unitsByName,
		results:     make(map[string]*UnitResult),
	}
}

// Run executes all units with proper trigger handling (one-time run)
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Println("Starting orchestrator...")
	o.checkAndExecuteTriggers(ctx, true)
	log.Println("Orchestrator finished")
	return nil
}

// RunDaemon executes in daemon mode, continuously checking triggers
func (o *Orchestrator) RunDaemon(ctx context.Context) error {
	log.Println("Starting orchestrator in daemon mode...")

	// Check interval - check triggers every 10 seconds as per README
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Run once immediately on startup (check all triggers including boot triggers)
	o.checkAndExecuteTriggers(ctx, true)

	for {
		select {
		case <-ctx.Done():
			log.Println("Orchestrator daemon shutting down...")
			return ctx.Err()
		case <-ticker.C:
			// During polling, skip startup triggers like boot triggers
			o.checkAndExecuteTriggers(ctx, false)
		}
	}
}

// checkAndExecuteTriggers checks all trigger units and executes them if they should fire
// If isStartup is true, all triggers are checked. If false, startup triggers are skipped.
func (o *Orchestrator) checkAndExecuteTriggers(ctx context.Context, isStartup bool) {
	// Clear results at the start of each check cycle to allow units to be re-executed
	// in subsequent trigger cycles (e.g., cron triggers firing every minute)
	o.results = make(map[string]*UnitResult)

	for _, unit := range o.units {
		if trigger, ok := unit.(TriggerUnit); ok {
			// Skip startup-only triggers during polling (only check them on app startup)
			if !isStartup && (unit.Type() == "trigger.boot" || unit.Type() == "trigger.start") {
				continue
			}

			shouldTrigger, err := trigger.Check(ctx)
			if err != nil {
				log.Printf("Error checking trigger '%s': %v", unit.Name(), err)
				continue
			}

			if shouldTrigger {
				log.Printf("Trigger '%s' activated", unit.Name())
				if err := o.executeUnit(ctx, unit); err != nil {
					log.Printf("Trigger '%s' failed: %v", unit.Name(), err)
				}
			}
		}
	}
}

// executeUnit runs a single unit and processes its triggers
func (o *Orchestrator) executeUnit(ctx context.Context, unit Unit) error {
	result := &UnitResult{
		Unit: unit,
	}

	// Capture output while also displaying it
	var outputBuf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create a pipe to capture output
	r, w, _ := os.Pipe()

	// Redirect stdout and stderr
	os.Stdout = w
	os.Stderr = w

	// Tee: copy to both buffer (for capturing) and original stdout (for display)
	done := make(chan bool)
	go func() {
		// Use MultiWriter to write to both buffer and original stdout
		mw := io.MultiWriter(&outputBuf, oldStdout)
		_, err := io.Copy(mw, r)
		if err != nil {
			log.Println("Error copying output buffer: ", err)
		}
		done <- true
	}()

	// Run the unit
	err := unit.Run(ctx)
	result.Error = err

	// Close writer and wait for copy to complete
	w.Close()
	<-done

	// Restore stdout/stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	result.Output = outputBuf.String()

	// Store result
	o.results[unit.Name()] = result

	// Process triggers for all units (not just TriggerUnits)
	o.processTriggers(ctx, unit, err, result.Output)

	return err
}

// processTriggers handles on_success, on_failure, and always triggers
// This works for both TriggerUnit and regular Unit types
func (o *Orchestrator) processTriggers(ctx context.Context, unit Unit, execErr error, output string) {
	var toTrigger []string

	// Check if this unit has trigger capabilities (on_success, on_failure, always)
	// Both TriggerUnit and regular units can have these fields
	switch u := unit.(type) {
	case TriggerUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	case *RunUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	case *RebootUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	case *LogUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	case *CountUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	case *EmailUnit:
		if execErr == nil {
			toTrigger = append(toTrigger, u.OnSuccess()...)
		} else {
			toTrigger = append(toTrigger, u.OnFailure()...)
		}
		toTrigger = append(toTrigger, u.Always()...)
	}

	// Execute triggered units
	for _, unitName := range toTrigger {
		targetUnit, ok := o.unitsByName[unitName]
		if !ok {
			log.Printf("Warning: referenced unit '%s' not found", unitName)
			continue
		}

		// If it's a log unit, pass the output and triggering unit name
		if logUnit, ok := targetUnit.(*LogUnit); ok {
			logUnit.SetOutput(output)
			logUnit.SetTriggeringUnit(unit.Name())
		}

		// If it's a count unit, pass the triggering unit name
		if countUnit, ok := targetUnit.(*CountUnit); ok {
			countUnit.SetTriggeringUnit(unit.Name())
		}

		// If it's an email unit, pass the output, triggering unit name, and error
		if emailUnit, ok := targetUnit.(*EmailUnit); ok {
			emailUnit.SetOutput(output)
			emailUnit.SetTriggeringUnit(unit.Name())
			emailUnit.SetTriggerError(execErr)
		}

		// Check if already executed in this trigger chain (prevents circular dependencies)
		if _, executed := o.results[unitName]; executed {
			log.Printf("Unit '%s' already executed in this chain, skipping to prevent circular dependency", unitName)
			continue
		}

		log.Printf("Triggering unit '%s'", unitName)
		if err := o.executeUnit(ctx, targetUnit); err != nil {
			log.Printf("Triggered unit '%s' failed: %v", unitName, err)
		}
	}
}

// RunSingleUnit executes a single unit by name
// If runTriggers is true, the unit runs and all its triggers are executed
// If runTriggers is false, the unit runs in isolation without executing its triggers
func (o *Orchestrator) RunSingleUnit(ctx context.Context, unitName string, runTriggers bool) error {
	unit, ok := o.unitsByName[unitName]
	if !ok {
		return fmt.Errorf("unit '%s' not found", unitName)
	}

	log.Printf("Executing single unit '%s'...", unitName)

	// Clear results
	o.results = make(map[string]*UnitResult)

	if runTriggers {
		// For trigger units, check if the trigger condition is met first
		if triggerUnit, ok := unit.(TriggerUnit); ok {
			shouldTrigger, err := triggerUnit.Check(ctx)
			if err != nil {
				log.Printf("Error checking trigger '%s': %v", unitName, err)
				return err
			}
			if !shouldTrigger {
				log.Printf("Trigger '%s' condition not met, skipping execution", unitName)
				return nil
			}
			log.Printf("Trigger '%s' condition met, executing...", unitName)
		}

		// Execute unit with triggers (normal execution)
		if err := o.executeUnit(ctx, unit); err != nil {
			log.Printf("Unit '%s' failed: %v", unitName, err)
			return err
		}
	} else {
		// Execute unit without triggers (for debugging)
		if err := o.executeUnitNoTriggers(ctx, unit); err != nil {
			log.Printf("Unit '%s' failed: %v", unitName, err)
			return err
		}
	}

	log.Printf("Unit '%s' completed", unitName)
	return nil
}

// executeUnitNoTriggers runs a single unit without processing its triggers
func (o *Orchestrator) executeUnitNoTriggers(ctx context.Context, unit Unit) error {
	result := &UnitResult{
		Unit: unit,
	}

	// Capture output while also displaying it
	var outputBuf bytes.Buffer
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create a pipe to capture output
	r, w, _ := os.Pipe()

	// Redirect stdout and stderr
	os.Stdout = w
	os.Stderr = w

	// Tee: copy to both buffer (for capturing) and original stdout (for display)
	done := make(chan bool)
	go func() {
		// Use MultiWriter to write to both buffer and original stdout
		mw := io.MultiWriter(&outputBuf, oldStdout)
		_, err := io.Copy(mw, r)
		if err != nil {
			log.Println("Error copying output buffer: ", err)
		}
		done <- true
	}()

	// Run the unit
	err := unit.Run(ctx)
	result.Error = err

	// Close writer and wait for copy to complete
	w.Close()
	<-done

	// Restore stdout/stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	result.Output = outputBuf.String()

	// Store result
	o.results[unit.Name()] = result

	// Do NOT process triggers in this method

	return err
}

// GetResults returns all execution results
func (o *Orchestrator) GetResults() map[string]*UnitResult {
	return o.results
}
