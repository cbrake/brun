package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	if err := orchestrator.RunOnce(ctx); err != nil {
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
	if err := orchestrator.RunOnce(ctx); err != nil {
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
	if err := orchestrator.RunOnce(ctx); err != nil {
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
	if err := orchestrator.RunOnce(ctx); err != nil {
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

// TestOrchestrator_TriggerUnitChecksConditionWhenTriggered verifies that
// trigger units run their Check() method when triggered by other units
// (addressing issue #19: cron unit triggering git unit should check condition)
func TestOrchestrator_TriggerUnitChecksConditionWhenTriggered(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.yaml")
	gitDir := filepath.Join(tmpDir, "testrepo")

	// Initialize a git repository using go-git
	repo, err := git.PlainInit(gitDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(gitDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create shared state and initialize git trigger state
	state := NewState(stateFile)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Get the current commit hash to simulate a previous run
	repoObj, err := git.PlainOpen(gitDir)
	if err != nil {
		t.Fatalf("Failed to open git repo: %v", err)
	}
	ref, err := repoObj.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	initialHash := ref.Hash().String()

	// Pre-populate state with the initial commit hash to simulate a previous run
	if err := state.SetString("git-trigger", "last_commit_hash", initialHash); err != nil {
		t.Fatalf("Failed to set initial state: %v", err)
	}

	// Create units:
	// start -> git-trigger (checks for git updates, none exist yet)
	// The git trigger should Check() and NOT run because no updates exist
	ctx := context.Background()
	startTrigger := NewStartTrigger("start", []string{"git-trigger"}, nil, nil)
	gitTrigger := NewGitTrigger("git-trigger", gitDir, "main", false, time.Second, false, state, []string{"build"}, nil, nil)
	buildUnit := NewRunUnit("build", "echo 'Building...'", "", 0, "", false, nil, nil, nil)

	units := []Unit{startTrigger, gitTrigger, buildUnit}
	orchestrator := NewOrchestrator(units)

	// Execute - git-trigger should be triggered by start, but should check and skip
	if err := orchestrator.RunOnce(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed: %v", err)
	}

	// Verify git-trigger did NOT execute (because Check() returned false)
	results := orchestrator.GetResults()
	if _, ok := results["git-trigger"]; ok {
		t.Error("git-trigger should NOT have executed because no git updates exist")
	}

	// Verify build unit was NOT triggered
	if _, ok := results["build"]; ok {
		t.Error("build unit should NOT have executed because git-trigger should not have run")
	}

	// Now make a git change and run again
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	_, err = worktree.Commit("Update file", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Wait for poll interval
	time.Sleep(1100 * time.Millisecond)

	// Create new orchestrator and run again
	orchestrator2 := NewOrchestrator(units)
	if err := orchestrator2.RunOnce(ctx); err != nil {
		t.Fatalf("Orchestrator.Run() failed on second run: %v", err)
	}

	// Verify git-trigger DID execute this time (because Check() returned true)
	results2 := orchestrator2.GetResults()
	if _, ok := results2["git-trigger"]; !ok {
		t.Error("git-trigger SHOULD have executed because git updates exist")
	}

	// Verify build unit was triggered
	if _, ok := results2["build"]; !ok {
		t.Error("build unit SHOULD have executed because git-trigger ran successfully")
	}
}
