package simpleci

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogConfig represents the configuration for a Log unit
type LogConfig struct {
	UnitConfig `yaml:",inline"`
	File       string `yaml:"file"`
}

// LogUnit writes log messages to a file
type LogUnit struct {
	name            string
	file            string
	output          string // Output from the triggering unit
	triggeringUnit  string // Name of the unit that triggered this log
	onSuccess       []string
	onFailure       []string
	always          []string
}

// NewLogUnit creates a new Log unit
func NewLogUnit(name, file string, onSuccess, onFailure, always []string) *LogUnit {
	return &LogUnit{
		name:      name,
		file:      file,
		onSuccess: onSuccess,
		onFailure: onFailure,
		always:    always,
	}
}

// Name returns the unit name
func (l *LogUnit) Name() string {
	return l.name
}

// Type returns the unit type
func (l *LogUnit) Type() string {
	return "log"
}

// SetOutput sets the output data from the triggering unit
func (l *LogUnit) SetOutput(output string) {
	l.output = output
}

// SetTriggeringUnit sets the name of the unit that triggered this log
func (l *LogUnit) SetTriggeringUnit(unitName string) {
	l.triggeringUnit = unitName
}

// Run executes the log unit
func (l *LogUnit) Run(ctx context.Context) error {
	log.Printf("Running log unit '%s'", l.name)

	// Create directory if it doesn't exist
	dir := filepath.Dir(l.file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(l.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Write log entry
	var logEntry string
	unitName := l.triggeringUnit
	if unitName == "" {
		unitName = "unknown"
	}

	timestamp := time.Now().Format(time.RFC3339)

	if l.output != "" {
		// Write the captured output from the triggering unit
		logEntry = fmt.Sprintf("=== Unit '%s' - %s ===\n%s\n", unitName, timestamp, l.output)
	} else {
		// Fallback if no output was captured
		logEntry = fmt.Sprintf("=== Unit '%s' - %s (no output) ===\n", unitName, timestamp)
	}

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	log.Printf("Log unit '%s' completed, wrote to %s", l.name, l.file)
	return nil
}

// OnSuccess returns the list of units to trigger on success
func (l *LogUnit) OnSuccess() []string {
	return l.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (l *LogUnit) OnFailure() []string {
	return l.onFailure
}

// Always returns the list of units to always trigger
func (l *LogUnit) Always() []string {
	return l.always
}
