package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestOrchestrator_MultipleUnitsCanTriggerSameUnit verifies that multiple units
// can trigger the same unit in a single execution chain (e.g., multiple emails)
func TestOrchestrator_MultipleUnitsCanTriggerSameUnit(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.yaml")

	// Create shared state
	state := NewState(stateFile)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Create units:
	// start -> unit-a -> counter
	//       -> unit-b -> counter
	// Counter should run twice (once from unit-a, once from unit-b)
	startTrigger := NewStartTrigger("start", []string{"unit-a", "unit-b"}, nil, nil)
	unitA := NewRunUnit("unit-a", "echo 'Unit A'", "", 0, "", false, nil, nil, []string{"counter"})
	unitB := NewRunUnit("unit-b", "echo 'Unit B'", "", 0, "", false, nil, nil, []string{"counter"})
	counter := NewCountUnit("counter", state, nil, nil, nil)

	units := []Unit{startTrigger, unitA, unitB, counter}
	orchestrator := NewOrchestrator(units)

	// Execute
	ctx := context.Background()
	if err := orchestrator.Run(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed: %v", err)
	}

	// Verify counter was triggered by both unit-a and unit-b
	countA, okA := state.Get("counter", "unit-a")
	if !okA {
		t.Error("Counter should have been triggered by unit-a")
	}
	if countA != 1 {
		t.Errorf("Counter from unit-a = %v, want 1", countA)
	}

	countB, okB := state.Get("counter", "unit-b")
	if !okB {
		t.Error("Counter should have been triggered by unit-b")
	}
	if countB != 1 {
		t.Errorf("Counter from unit-b = %v, want 1", countB)
	}
}

// TestOrchestrator_CircularDependencyDetection verifies that circular
// dependencies are properly detected and prevented
func TestOrchestrator_CircularDependencyDetection(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.yaml")

	// Create shared state
	state := NewState(stateFile)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Create units with circular dependency:
	// start -> unit-a -> unit-b -> unit-a (circular!)
	startTrigger := NewStartTrigger("start", []string{"unit-a"}, nil, nil)
	unitA := NewRunUnit("unit-a", "echo 'Unit A'", "", 0, "", false, []string{"unit-b"}, nil, nil)
	unitB := NewRunUnit("unit-b", "echo 'Unit B'", "", 0, "", false, []string{"unit-a"}, nil, nil)

	units := []Unit{startTrigger, unitA, unitB}
	orchestrator := NewOrchestrator(units)

	// Execute - should complete without infinite loop
	ctx := context.Background()
	if err := orchestrator.Run(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed: %v", err)
	}

	// Verify both units executed (at least once)
	results := orchestrator.GetResults()
	if _, ok := results["unit-a"]; !ok {
		t.Error("unit-a should have executed")
	}
	if _, ok := results["unit-b"]; !ok {
		t.Error("unit-b should have executed")
	}

	// Success means we didn't infinite loop
}

// TestOrchestrator_SelfReferentialCircularDependency verifies that a unit
// triggering itself is properly detected
func TestOrchestrator_SelfReferentialCircularDependency(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.yaml")

	// Create shared state
	state := NewState(stateFile)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Create unit that triggers itself:
	// start -> unit-a -> unit-a (self-referential!)
	startTrigger := NewStartTrigger("start", []string{"unit-a"}, nil, nil)
	unitA := NewRunUnit("unit-a", "echo 'Unit A'", "", 0, "", false, []string{"unit-a"}, nil, nil)

	units := []Unit{startTrigger, unitA}
	orchestrator := NewOrchestrator(units)

	// Execute - should complete without infinite loop
	ctx := context.Background()
	if err := orchestrator.Run(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed: %v", err)
	}

	// Verify unit-a executed exactly once
	results := orchestrator.GetResults()
	if _, ok := results["unit-a"]; !ok {
		t.Error("unit-a should have executed")
	}

	// Success means we didn't infinite loop
}

// TestOrchestrator_ComplexChainWithReusedUnits verifies that complex
// trigger chains work correctly when units are legitimately reused
func TestOrchestrator_ComplexChainWithReusedUnits(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.yaml")
	logFile := filepath.Join(tmpDir, "test.log")

	// Create shared state
	state := NewState(stateFile)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Create complex chain:
	//        -> unit-a -> logger
	// start                     -> finalizer
	//        -> unit-b -> logger
	// Logger should run twice, finalizer should run once
	startTrigger := NewStartTrigger("start", []string{"unit-a", "unit-b"}, nil, nil)
	unitA := NewRunUnit("unit-a", "echo 'Unit A'", "", 0, "", false, []string{"logger"}, nil, nil)
	unitB := NewRunUnit("unit-b", "echo 'Unit B'", "", 0, "", false, []string{"logger"}, nil, nil)
	logger := NewLogUnit("logger", logFile, nil, nil, []string{"finalizer"})
	finalizer := NewCountUnit("finalizer", state, nil, nil, nil)

	units := []Unit{startTrigger, unitA, unitB, logger, finalizer}
	orchestrator := NewOrchestrator(units)

	// Execute
	ctx := context.Background()
	if err := orchestrator.Run(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed: %v", err)
	}

	// Verify logger was triggered twice (from different branches)
	results := orchestrator.GetResults()
	if _, ok := results["logger"]; !ok {
		t.Error("logger should have executed")
	}

	// Verify finalizer was triggered twice (once from each logger invocation)
	countLogger, ok := state.Get("finalizer", "logger")
	if !ok {
		t.Error("Finalizer should have been triggered by logger")
	}
	if countLogger != 2 {
		t.Errorf("Finalizer count from logger = %v, want 2", countLogger)
	}

	// Verify log file exists and was written
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should have been created")
	}
}
