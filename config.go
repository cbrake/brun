package metalci

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigBlock represents the config section of the configuration file
type ConfigBlock struct {
	StateLocation string `yaml:"state_location"`
}

// Config represents the SimplCI configuration file
type Config struct {
	ConfigBlock ConfigBlock         `yaml:"config"`
	Units       []UnitConfigWrapper `yaml:"units"`
}

// UnitConfigWrapper wraps different unit configuration types
type UnitConfigWrapper struct {
	Start  *StartConfig  `yaml:"start,omitempty"`
	Boot   *BootConfig   `yaml:"boot,omitempty"`
	Reboot *RebootConfig `yaml:"reboot,omitempty"`
	Run    *RunConfig    `yaml:"run,omitempty"`
	Log    *LogConfig    `yaml:"log,omitempty"`
	// Future trigger types can be added here
	// Git  *GitConfig  `yaml:"git,omitempty"`
	// Cron *CronConfig `yaml:"cron,omitempty"`
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
	// Validate required fields
	if c.ConfigBlock.StateLocation == "" {
		return nil, fmt.Errorf("config.state_location is required in config file")
	}

	// Create shared state manager
	state := NewState(c.ConfigBlock.StateLocation)

	var units []Unit

	for i, wrapper := range c.Units {
		if wrapper.Start != nil {
			cfg := wrapper.Start
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}

			unit := NewStartTrigger(
				cfg.Name,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Boot != nil {
			cfg := wrapper.Boot
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}

			unit := NewBootTrigger(
				cfg.Name,
				state,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Reboot != nil {
			cfg := wrapper.Reboot
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}

			unit := NewRebootUnit(
				cfg.Name,
				cfg.Delay,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Run != nil {
			cfg := wrapper.Run
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if cfg.Script == "" {
				return nil, fmt.Errorf("unit %d: script is required", i)
			}

			unit := NewRunUnit(
				cfg.Name,
				cfg.Script,
				cfg.Directory,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Log != nil {
			cfg := wrapper.Log
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if cfg.File == "" {
				return nil, fmt.Errorf("unit %d: file is required", i)
			}

			unit := NewLogUnit(
				cfg.Name,
				cfg.File,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}
		// Add other unit types here as they are implemented
	}

	return units, nil
}
