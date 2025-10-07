package brun

import (
	"fmt"
	"os"
	"time"

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
	Count  *CountConfig  `yaml:"count,omitempty"`
	Cron   *CronConfig   `yaml:"cron,omitempty"`
	Email  *EmailConfig  `yaml:"email,omitempty"`
	File   *FileConfig   `yaml:"file,omitempty"`
	Git    *GitConfig    `yaml:"git,omitempty"`
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

			// Parse timeout if specified
			var timeout time.Duration
			if cfg.Timeout != "" {
				var err error
				timeout, err = time.ParseDuration(cfg.Timeout)
				if err != nil {
					return nil, fmt.Errorf("unit %d (%s): invalid timeout format '%s': %w", i, cfg.Name, cfg.Timeout, err)
				}
			}

			unit := NewRunUnit(
				cfg.Name,
				cfg.Script,
				cfg.Directory,
				timeout,
				cfg.Shell,
				cfg.UsePTY,
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

		if wrapper.Count != nil {
			cfg := wrapper.Count
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}

			unit := NewCountUnit(
				cfg.Name,
				state,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Cron != nil {
			cfg := wrapper.Cron
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if cfg.Schedule == "" {
				return nil, fmt.Errorf("unit %d: schedule is required", i)
			}

			unit := NewCronTrigger(
				cfg.Name,
				cfg.Schedule,
				state,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Email != nil {
			cfg := wrapper.Email
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if len(cfg.To) == 0 {
				return nil, fmt.Errorf("unit %d: to is required", i)
			}
			if cfg.From == "" {
				return nil, fmt.Errorf("unit %d: from is required", i)
			}
			if cfg.SMTPHost == "" {
				return nil, fmt.Errorf("unit %d: smtp_host is required", i)
			}

			// Set defaults
			smtpPort := cfg.SMTPPort
			if smtpPort == 0 {
				smtpPort = 587 // Default to submission port
			}

			smtpUseTLS := true
			if cfg.SMTPUseTLS != nil {
				smtpUseTLS = *cfg.SMTPUseTLS
			}

			includeOutput := true
			if cfg.IncludeOutput != nil {
				includeOutput = *cfg.IncludeOutput
			}

			unit := NewEmailUnit(
				cfg.Name,
				cfg.To,
				cfg.From,
				cfg.SubjectPrefix,
				cfg.SMTPHost,
				smtpPort,
				cfg.SMTPUser,
				cfg.SMTPPassword,
				smtpUseTLS,
				includeOutput,
				cfg.LimitLines,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.File != nil {
			cfg := wrapper.File
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if cfg.Pattern == "" {
				return nil, fmt.Errorf("unit %d: pattern is required", i)
			}

			unit := NewFileTrigger(
				cfg.Name,
				cfg.Pattern,
				state,
				cfg.OnSuccess,
				cfg.OnFailure,
				cfg.Always,
			)
			units = append(units, unit)
		}

		if wrapper.Git != nil {
			cfg := wrapper.Git
			if cfg.Name == "" {
				return nil, fmt.Errorf("unit %d: name is required", i)
			}
			if cfg.Repository == "" {
				return nil, fmt.Errorf("unit %d: repository is required", i)
			}
			if cfg.Branch == "" {
				return nil, fmt.Errorf("unit %d: branch is required", i)
			}

			unit := NewGitTrigger(
				cfg.Name,
				cfg.Repository,
				cfg.Branch,
				cfg.Reset,
				state,
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
