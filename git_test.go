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

func TestGitTrigger_Check(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Add and commit
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	commit, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create git trigger
	trigger := NewGitTrigger(
		"test-git",
		repoPath,
		"main",
		false,       // reset
		time.Second, // poll interval (1 second for testing)
		false,       // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger (new repository)
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first check")
	}

	// Second check should not trigger (no new commits)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger on second check (no new commits)")
	}

	// Make a new commit
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Wait for poll interval to pass
	time.Sleep(1100 * time.Millisecond)

	// Third check should trigger (new commit)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger after new commit")
	}

	// Verify commit hash was stored
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	storedHash, ok := state.GetString("test-git", "last_commit_hash")
	if !ok {
		t.Error("Expected commit hash to be stored in state")
	}

	// Verify it's the latest commit hash
	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	if storedHash != ref.Hash().String() {
		t.Errorf("Expected stored hash %s, got %s", ref.Hash().String(), storedHash)
	}

	// Verify first commit hash is different from second
	if commit.String() == ref.Hash().String() {
		t.Error("First and second commit hashes should be different")
	}
}

func TestGitTrigger_Run(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	_, err = worktree.Commit("Test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create state
	state := NewState(stateFile)

	// Create git trigger
	trigger := NewGitTrigger(
		"test-git-run",
		repoPath,
		"main",
		false,         // reset
		2*time.Minute, // poll interval
		false,         // debug
		state,
		[]string{"build"},
		[]string{"error"},
		[]string{"always"},
	)

	if trigger.Name() != "test-git-run" {
		t.Errorf("Expected name 'test-git-run', got '%s'", trigger.Name())
	}

	if trigger.Type() != "trigger.git" {
		t.Errorf("Expected type 'trigger.git', got '%s'", trigger.Type())
	}

	ctx := context.Background()
	if err := trigger.Run(ctx); err != nil {
		t.Errorf("Run failed: %v", err)
	}

	onSuccess := trigger.OnSuccess()
	if len(onSuccess) != 1 || onSuccess[0] != "build" {
		t.Errorf("Expected on_success [build], got %v", onSuccess)
	}

	onFailure := trigger.OnFailure()
	if len(onFailure) != 1 || onFailure[0] != "error" {
		t.Errorf("Expected on_failure [error], got %v", onFailure)
	}

	always := trigger.Always()
	if len(always) != 1 || always[0] != "always" {
		t.Errorf("Expected always [always], got %v", always)
	}
}

func TestLoadConfig_WithGitUnit(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	stateFile := filepath.Join(tempDir, "state.yaml")
	repoPath := filepath.Join(tempDir, "repo")

	// Initialize a git repository
	_, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	configContent := `config:
  state_location: ` + stateFile + `

units:
  - git:
      name: watch-repo
      repository: ` + repoPath + `
      branch: main
      on_success:
        - build
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	units, err := config.CreateUnits()
	if err != nil {
		t.Fatalf("CreateUnits failed: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(units))
	}

	unit := units[0]
	if unit.Name() != "watch-repo" {
		t.Errorf("Expected name 'watch-repo', got '%s'", unit.Name())
	}

	if unit.Type() != "trigger.git" {
		t.Errorf("Expected type 'trigger.git', got '%s'", unit.Type())
	}

	gitTrigger, ok := unit.(*GitTrigger)
	if !ok {
		t.Fatal("Unit is not a GitTrigger")
	}

	if gitTrigger.repository != repoPath {
		t.Errorf("Expected repository '%s', got '%s'", repoPath, gitTrigger.repository)
	}

	if len(gitTrigger.onSuccess) != 1 || gitTrigger.onSuccess[0] != "build" {
		t.Errorf("Expected on_success [build], got %v", gitTrigger.onSuccess)
	}
}

func TestCreateUnits_GitMissingRepository(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")

	config := &Config{
		ConfigBlock: ConfigBlock{
			StateLocation: stateFile,
		},
		Units: []UnitConfigWrapper{
			{
				Git: &GitConfig{
					UnitConfig: UnitConfig{
						Name: "test",
					},
					// Repository is missing
				},
			},
		},
	}

	_, err := config.CreateUnits()
	if err == nil {
		t.Error("Expected error for missing repository")
	}
}

func TestGitTrigger_InvalidRepository(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.yaml")
	invalidRepo := filepath.Join(tempDir, "nonexistent")

	state := NewState(stateFile)
	trigger := NewGitTrigger(
		"test-invalid",
		invalidRepo,
		"main",
		false,         // reset
		2*time.Minute, // poll interval
		false,         // debug
		state,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	_, err := trigger.Check(ctx, CheckModePolling)
	if err == nil {
		t.Error("Expected error for invalid repository")
	}
}

// TestGitTrigger_CheckModePolling_NoPollInterval tests passive mode during polling
// Git unit with pollInterval=0 should NOT check during orchestrator polling
func TestGitTrigger_CheckModePolling_NoPollInterval(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with NO poll interval (passive mode)
	trigger := NewGitTrigger(
		"test-git-passive",
		repoPath,
		"main",
		false, // reset
		0,     // NO poll interval - passive mode
		false, // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// Check with CheckModePolling should return false without checking git
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger in CheckModePolling with pollInterval=0")
	}

	// Verify that state was NOT updated (git was not checked)
	if err := state.Load(); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to load state: %v", err)
	}
	_, ok := state.GetString("test-git-passive", "last_commit_hash")
	if ok {
		t.Error("Expected no commit hash in state (git should not have been checked)")
	}
}

// TestGitTrigger_CheckModeManual_NoPollInterval tests passive mode when manually triggered
// Git unit with pollInterval=0 SHOULD check when triggered by another unit
func TestGitTrigger_CheckModeManual_NoPollInterval(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with NO poll interval (passive mode)
	trigger := NewGitTrigger(
		"test-git-passive-manual",
		repoPath,
		"main",
		false, // reset
		0,     // NO poll interval - passive mode
		false, // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// Check with CheckModeManual should check git and return true (first time)
	shouldTrigger, err := trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger in CheckModeManual (first check of repo)")
	}

	// Verify that state WAS updated (git was checked)
	if err := state.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}
	storedHash, ok := state.GetString("test-git-passive-manual", "last_commit_hash")
	if !ok {
		t.Error("Expected commit hash to be stored in state")
	}

	// Verify it's the correct commit hash
	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	if storedHash != ref.Hash().String() {
		t.Errorf("Expected stored hash %s, got %s", ref.Hash().String(), storedHash)
	}

	// Second check should not trigger (no new commits)
	shouldTrigger, err = trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger on second CheckModeManual (no new commits)")
	}
}

// TestGitTrigger_CheckModePolling_IntervalNotElapsed tests active mode polling with interval not elapsed
// Git unit should NOT check if poll interval hasn't passed
func TestGitTrigger_CheckModePolling_IntervalNotElapsed(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with 5 second poll interval
	trigger := NewGitTrigger(
		"test-git-interval",
		repoPath,
		"main",
		false,         // reset
		5*time.Second, // poll interval
		false,         // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger (new repository)
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first CheckModePolling")
	}

	// Make a new commit
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Immediate second check (before interval elapsed) should NOT trigger
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if shouldTrigger {
		t.Error("Expected no trigger in CheckModePolling before interval elapsed")
	}
}

// TestGitTrigger_CheckModePolling_IntervalElapsed tests active mode polling with interval elapsed
// Git unit SHOULD check when poll interval has passed
func TestGitTrigger_CheckModePolling_IntervalElapsed(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with 1 second poll interval (short for testing)
	trigger := NewGitTrigger(
		"test-git-elapsed",
		repoPath,
		"main",
		false,         // reset
		1*time.Second, // short poll interval for testing
		false,         // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first CheckModePolling")
	}

	// Make a new commit
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Wait for poll interval to pass
	time.Sleep(1100 * time.Millisecond)

	// Check after interval should trigger (new commit detected)
	shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger in CheckModePolling after interval elapsed with new commit")
	}
}

// TestGitTrigger_CheckModeManual_IgnoresPollInterval tests that manual mode ignores poll interval
// Git unit with pollInterval should check immediately when in CheckModeManual
func TestGitTrigger_CheckModeManual_IgnoresPollInterval(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with long poll interval
	trigger := NewGitTrigger(
		"test-git-manual-ignore-interval",
		repoPath,
		"main",
		false,          // reset
		10*time.Minute, // long poll interval
		false,          // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First manual check should trigger (new repository)
	shouldTrigger, err := trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first CheckModeManual")
	}

	// Make a new commit
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Immediate second CheckModeManual should trigger (ignoring poll interval)
	shouldTrigger, err = trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger in CheckModeManual immediately (should ignore poll interval)")
	}
}

// TestGitTrigger_MultipleCheckModePolling tests that multiple polling checks respect interval
func TestGitTrigger_MultipleCheckModePolling(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with 2 second poll interval
	trigger := NewGitTrigger(
		"test-git-multiple-polling",
		repoPath,
		"main",
		false,         // reset
		2*time.Second, // poll interval
		false,         // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger
	shouldTrigger, err := trigger.Check(ctx, CheckModePolling)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first check")
	}

	// Multiple rapid checks should not trigger (interval not elapsed)
	for i := 0; i < 3; i++ {
		shouldTrigger, err = trigger.Check(ctx, CheckModePolling)
		if err != nil {
			t.Fatalf("Check %d failed: %v", i+2, err)
		}
		if shouldTrigger {
			t.Errorf("Check %d: Expected no trigger (interval not elapsed)", i+2)
		}
	}
}

// TestGitTrigger_MultipleCheckModeManual tests that multiple manual checks always check
func TestGitTrigger_MultipleCheckModeManual(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	stateFile := filepath.Join(tempDir, "state.yaml")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create and commit a test file
	testFile := filepath.Join(repoPath, "test.txt")
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

	// Create state
	state := NewState(stateFile)

	// Create git trigger with long poll interval
	trigger := NewGitTrigger(
		"test-git-multiple-manual",
		repoPath,
		"main",
		false,          // reset
		10*time.Minute, // long poll interval
		false,          // debug
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First manual check should trigger
	shouldTrigger, err := trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first CheckModeManual")
	}

	// Multiple rapid manual checks should all check git (but won't trigger without new commits)
	checkCount := 0
	for i := 0; i < 3; i++ {
		shouldTrigger, err = trigger.Check(ctx, CheckModeManual)
		if err != nil {
			t.Fatalf("Check %d failed: %v", i+2, err)
		}
		// Should not trigger (no new commits), but git WAS checked each time
		if shouldTrigger {
			t.Errorf("Check %d: Expected no trigger (no new commits)", i+2)
		}
		checkCount++
	}

	if checkCount != 3 {
		t.Errorf("Expected 3 checks, got %d", checkCount)
	}

	// Now make a new commit and verify manual check detects it immediately
	if err := os.WriteFile(testFile, []byte("updated content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if _, err := worktree.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Manual check should immediately detect the new commit
	shouldTrigger, err = trigger.Check(ctx, CheckModeManual)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger in CheckModeManual after new commit")
	}
}
