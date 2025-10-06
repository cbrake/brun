package brun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
		state,
		[]string{"build"},
		nil,
		nil,
	)

	ctx := context.Background()

	// First check should trigger (new repository)
	shouldTrigger, err := trigger.Check(ctx)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !shouldTrigger {
		t.Error("Expected trigger on first check")
	}

	// Second check should not trigger (no new commits)
	shouldTrigger, err = trigger.Check(ctx)
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

	// Third check should trigger (new commit)
	shouldTrigger, err = trigger.Check(ctx)
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
		state,
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	_, err := trigger.Check(ctx)
	if err == nil {
		t.Error("Expected error for invalid repository")
	}
}
