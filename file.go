package brun

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// FileTrigger is a trigger unit that fires when files matching a pattern change
type FileTrigger struct {
	name      string
	pattern   string
	state     *State
	onSuccess []string
	onFailure []string
	always    []string
}

// FileConfig represents the configuration for a file trigger
type FileConfig struct {
	UnitConfig `yaml:",inline"`
	Pattern    string `yaml:"pattern"`
}

// NewFileTrigger creates a new file trigger unit
func NewFileTrigger(name, pattern string, state *State, onSuccess, onFailure, always []string) *FileTrigger {
	return &FileTrigger{
		name:      name,
		pattern:   pattern,
		state:     state,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the name of the unit
func (f *FileTrigger) Name() string {
	return f.name
}

// Type returns the unit type
func (f *FileTrigger) Type() string {
	return "trigger.file"
}

// getFileHash computes SHA256 hash of a file
func (f *FileTrigger) getFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getFilesState returns a map of file paths to their hashes
func (f *FileTrigger) getFilesState() (map[string]string, error) {
	// Use doublestar for recursive glob support (supports **)
	// This works with both relative and absolute patterns
	matches, err := doublestar.FilepathGlob(f.pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob pattern '%s': %w", f.pattern, err)
	}

	filesState := make(map[string]string)
	for _, path := range matches {
		// Check if it's a regular file
		info, err := os.Stat(path)
		if err != nil {
			continue // Skip files we can't stat
		}
		if info.IsDir() {
			continue // Skip directories
		}

		hash, err := f.getFileHash(path)
		if err != nil {
			// If we can't hash a file, use empty string
			hash = ""
		}
		filesState[path] = hash
	}

	return filesState, nil
}

// filesStateToString converts file state map to a sortable string representation
func (f *FileTrigger) filesStateToString(filesState map[string]string) string {
	// Sort keys for consistent output
	keys := make([]string, 0, len(filesState))
	for k := range filesState {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", k, filesState[k]))
	}
	return strings.Join(parts, "|")
}

// Check returns true if files matching the pattern have changed
func (f *FileTrigger) Check(ctx context.Context) (bool, error) {
	// Get current files state
	currentState, err := f.getFilesState()
	if err != nil {
		return false, fmt.Errorf("failed to get current files state: %w", err)
	}

	// Load state
	if err := f.state.Load(); err != nil {
		return false, fmt.Errorf("failed to load state: %w", err)
	}

	// Convert current state to string
	currentStateStr := f.filesStateToString(currentState)

	// Get last state from state file
	lastStateStr, ok := f.state.GetString(f.name, "files_state")
	if !ok {
		// No previous state, this is the first run
		if err := f.state.SetString(f.name, "files_state", currentStateStr); err != nil {
			return false, fmt.Errorf("failed to save files state: %w", err)
		}
		return true, nil
	}

	// Check if state has changed
	if currentStateStr != lastStateStr {
		// Files have changed, update state and trigger
		if err := f.state.SetString(f.name, "files_state", currentStateStr); err != nil {
			return false, fmt.Errorf("failed to save files state: %w", err)
		}
		return true, nil
	}

	return false, nil
}

// OnSuccess returns the list of units to trigger on success
func (f *FileTrigger) OnSuccess() []string {
	return f.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (f *FileTrigger) OnFailure() []string {
	return f.onFailure
}

// Always returns the list of units to trigger regardless of success/failure
func (f *FileTrigger) Always() []string {
	return f.always
}

// Run executes the trigger unit
func (f *FileTrigger) Run(ctx context.Context) error {
	triggered, err := f.Check(ctx)
	if err != nil {
		return err
	}

	if triggered {
		// Get current files for logging
		currentState, _ := f.getFilesState()
		fileCount := len(currentState)
		log.Printf("File trigger '%s' activated (%d file(s) matching '%s')", f.name, fileCount, f.pattern)
	}

	return nil
}
