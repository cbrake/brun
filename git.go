package brun

import (
	"context"
	"fmt"
	"log"

	"github.com/go-git/go-git/v5"
)

// GitTrigger is a trigger unit that fires when git repository changes are detected
type GitTrigger struct {
	name       string
	repository string
	state      *State
	onSuccess  []string
	onFailure  []string
	always     []string
}

// GitConfig represents the configuration for a git trigger
type GitConfig struct {
	UnitConfig `yaml:",inline"`
	Repository string `yaml:"repository"`
}

// NewGitTrigger creates a new git trigger unit
func NewGitTrigger(name, repository string, state *State, onSuccess, onFailure, always []string) *GitTrigger {
	return &GitTrigger{
		name:       name,
		repository: repository,
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
