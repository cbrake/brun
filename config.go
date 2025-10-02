package simpleci

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the SimplCI configuration file
type Config struct {
	Units []UnitConfigWrapper `yaml:"units"`
}

// UnitConfigWrapper wraps different unit configuration types
type UnitConfigWrapper struct {
	SystemBooted *SystemBootedConfig `yaml:"system_booted,omitempty"`
	// Future trigger types can be added here
	// Git          *GitConfig          `yaml:"git,omitempty"`
	// Cron         *CronConfig         `yaml:"cron,omitempty"`
}

// LoadConfig loads a configuration file from the given path
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// CreateUnits creates unit instances from the configuration
func (c *Config) CreateUnits() ([]Unit, error) {
	var units []Unit

	for i, wrapper := range c.Units {
		if wrapper.SystemBooted != nil {
			cfg := wrapper.SystemBooted
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}

			unit := NewSystemBootedTrigger(
				cfg.Name,
				cfg.StateFile,
				cfg.Trigger,
			)
			units = append(units, unit)
		}
		// Add other unit types here as they are implemented
	}

	return units, nil
}
