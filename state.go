package simpleci

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// State represents the common state file for all units
type State struct {
	filePath string
	data     map[string]any
}

// NewState creates a new state manager with the given file path
func NewState(filePath string) *State {
	if filePath == "" {
		filePath = "/var/lib/simpleci/state.yaml"
	}
	return &State{
		filePath: filePath,
		data:     make(map[string]any),
	}
}

// Load reads the state file from disk
func (s *State) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, start with empty state
			s.data = make(map[string]any)
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := yaml.Unmarshal(data, &s.data); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	return nil
}

// Save writes the state file to disk
func (s *State) Save() error {
	// Ensure directory exists
	dir := s.filePath[:strings.LastIndex(s.filePath, "/")]
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}
	}

	data, err := yaml.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Get retrieves a value from state for the given unit name and key
func (s *State) Get(unitName, key string) (any, bool) {
	unitData, ok := s.data[unitName]
	if !ok {
		return nil, false
	}

	unitMap, ok := unitData.(map[string]any)
	if !ok {
		return nil, false
	}

	value, ok := unitMap[key]
	return value, ok
}

// Set stores a value in state for the given unit name and key and automatically saves
func (s *State) Set(unitName, key string, value any) error {
	unitData, ok := s.data[unitName]
	if !ok {
		unitData = make(map[string]any)
		s.data[unitName] = unitData
	}

	unitMap, ok := unitData.(map[string]any)
	if !ok {
		unitMap = make(map[string]any)
		s.data[unitName] = unitMap
	}

	unitMap[key] = value

	// Automatically save after setting
	return s.Save()
}

// GetString retrieves a string value from state
func (s *State) GetString(unitName, key string) (string, bool) {
	value, ok := s.Get(unitName, key)
	if !ok {
		return "", false
	}

	str, ok := value.(string)
	return str, ok
}

// SetString stores a string value in state and automatically saves
func (s *State) SetString(unitName, key, value string) error {
	return s.Set(unitName, key, value)
}
