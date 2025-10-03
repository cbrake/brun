package brun

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
)

// UnitResult represents the result of a unit execution
type UnitResult struct {
	Unit   Unit
	Error  error
	Output string // Captured stdout/stderr
}

// Orchestrator manages unit execution and triggering
type Orchestrator struct {
	units      []Unit
	unitsByName map[string]Unit
	results    map[string]*UnitResult
}

// NewOrchestrator creates a new orchestrator with the given units
func NewOrchestrator(units []Unit) *Orchestrator {
	unitsByName := make(map[string]Unit)
	for _, unit := range units {
		unitsByName[unit.Name()] = unit
	}

	return &Orchestrator{
		units:      units,
		unitsByName: unitsByName,
		results:    make(map[string]*UnitResult),
	}
}

// Run executes all units with proper trigger handling
func (o *Orchestrator) Run(ctx context.Context) error {
	log.Println("Starting orchestrator...")

	// Check all trigger units and execute if they should fire
	for _, unit := range o.units {
		if trigger, ok := unit.(TriggerUnit); ok {
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

	log.Println("Orchestrator finished")
	return nil
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
		io.Copy(mw, r)
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

		// Check if already executed
		if _, executed := o.results[unitName]; executed {
			log.Printf("Unit '%s' already executed, skipping", unitName)
			continue
		}

		log.Printf("Triggering unit '%s'", unitName)
		if err := o.executeUnit(ctx, targetUnit); err != nil {
			log.Printf("Triggered unit '%s' failed: %v", unitName, err)
		}
	}
}

// GetResults returns all execution results
func (o *Orchestrator) GetResults() map[string]*UnitResult {
	return o.results
}
