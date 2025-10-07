package brun

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/go-git/go-git/v5"
)

// GitTrigger is a trigger unit that fires when git repository changes are detected
type GitTrigger struct {
	name       string
	repository string
	branch     string
	reset      bool
	state      *State
	onSuccess  []string
	onFailure  []string
	always     []string
}

// GitConfig represents the configuration for a git trigger
type GitConfig struct {
	UnitConfig `yaml:",inline"`
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Reset      bool   `yaml:"reset"`
}

// NewGitTrigger creates a new git trigger unit
func NewGitTrigger(name, repository, branch string, reset bool, state *State, onSuccess, onFailure, always []string) *GitTrigger {
	return &GitTrigger{
		name:       name,
		repository: repository,
		branch:     branch,
		reset:      reset,
		state:      state,
		onSuccess:  onSuccess,
		onFailure:  onFailure,
		always:     always,
	}
}

// Name returns the name of the unit
func (g *GitTrigger) Name() string {
	return g.name
}

// Type returns the unit type
func (g *GitTrigger) Type() string {
	return "trigger.git"
}

// isLocalWorkspace checks if the repository path is a local Git workspace
func (g *GitTrigger) isLocalWorkspace() bool {
	// Try to open as a local repository
	_, err := git.PlainOpen(g.repository)
	return err == nil
}

// updateWorkspace updates a local Git workspace to the latest commit on the specified branch
// Uses native git commands for reliability with SSH, submodules, etc.
func (g *GitTrigger) updateWorkspace(ctx context.Context) error {
	// Verify repository exists using go-git
	repo, err := git.PlainOpen(g.repository)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Check if repository has remotes
	remotes, err := repo.Remotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	// If no remotes, skip update (local-only repository)
	if len(remotes) == 0 {
		return nil
	}

	// Use native git commands for the update operations
	log.Printf("Fetching updates for repository %s", g.repository)

	// git fetch origin
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetchCmd.Dir = g.repository
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch updates: %w\nOutput: %s", err, output)
	}

	// git checkout <branch>
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", g.branch)
	checkoutCmd.Dir = g.repository
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w\nOutput: %s", err, output)
	}

	// git reset --hard origin/<branch> (if reset enabled) or git merge origin/<branch>
	if g.reset {
		remoteBranch := fmt.Sprintf("origin/%s", g.branch)
		resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", remoteBranch)
		resetCmd.Dir = g.repository
		if output, err := resetCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to reset workspace: %w\nOutput: %s", err, output)
		}
		log.Printf("Reset workspace to %s", remoteBranch)
	} else {
		remoteBranch := fmt.Sprintf("origin/%s", g.branch)
		mergeCmd := exec.CommandContext(ctx, "git", "merge", remoteBranch)
		mergeCmd.Dir = g.repository
		if output, err := mergeCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to merge updates: %w\nOutput: %s", err, output)
		}
	}

	// git submodule update --init --recursive
	log.Printf("Updating submodules for repository %s", g.repository)
	submoduleCmd := exec.CommandContext(ctx, "git", "submodule", "update", "--init", "--recursive")
	submoduleCmd.Dir = g.repository
	if output, err := submoduleCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update submodules: %w\nOutput: %s", err, output)
	}

	return nil
}

// getCurrentCommitHash gets the current HEAD commit hash from the repository
func (g *GitTrigger) getCurrentCommitHash() (string, error) {
	// Open the repository
	repo, err := git.PlainOpen(g.repository)
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String(), nil
}

// Check returns true if the git repository has new commits since last check
func (g *GitTrigger) Check(ctx context.Context) (bool, error) {
	// If this is a local workspace, update it first
	if g.isLocalWorkspace() {
		if err := g.updateWorkspace(ctx); err != nil {
			return false, fmt.Errorf("failed to update workspace: %w", err)
		}
	}

	// Get current commit hash
	currentHash, err := g.getCurrentCommitHash()
	if err != nil {
		return false, fmt.Errorf("failed to check git repository: %w", err)
	}

	// Load state
	if err := g.state.Load(); err != nil {
		return false, fmt.Errorf("failed to load state: %w", err)
	}

	// Get last commit hash from state
	lastHash, ok := g.state.GetString(g.name, "last_commit_hash")
	if !ok {
		// No previous commit hash, this is the first run
		// Save current hash and trigger
		if err := g.state.SetString(g.name, "last_commit_hash", currentHash); err != nil {
			return false, fmt.Errorf("failed to save commit hash: %w", err)
		}
		return true, nil
	}

	// Check if commit hash has changed
	if currentHash != lastHash {
		// Repository has new commits, update state and trigger
		if err := g.state.SetString(g.name, "last_commit_hash", currentHash); err != nil {
			return false, fmt.Errorf("failed to save commit hash: %w", err)
		}
		return true, nil
	}

	return false, nil
}

// OnSuccess returns the list of units to trigger on success
func (g *GitTrigger) OnSuccess() []string {
	return g.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (g *GitTrigger) OnFailure() []string {
	return g.onFailure
}

// Always returns the list of units to trigger regardless of success/failure
func (g *GitTrigger) Always() []string {
	return g.always
}

// Run executes the trigger unit
func (g *GitTrigger) Run(ctx context.Context) error {
	triggered, err := g.Check(ctx)
	if err != nil {
		return err
	}

	if triggered {
		// Get current commit hash for logging
		currentHash, _ := g.getCurrentCommitHash()
		shortHash := currentHash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		log.Printf("Git trigger '%s' activated (commit: %s)", g.name, shortHash)
	}

	return nil
}
