package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Window represents a scheduled time range on specific days.
type Window struct {
	Start string   `yaml:"start" json:"start"`
	End   string   `yaml:"end"   json:"end"`
	Days  []string `yaml:"days"  json:"days"`
}

// IPC holds IPC transport configuration.
type IPC struct {
	PipeName string `yaml:"pipe_name"`
	HTTPPort int    `yaml:"http_port"`
}

// Log holds logging configuration.
type Log struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Config is the top-level configuration structure.
type Config struct {
	Mode        string   `yaml:"mode"`
	Schedule    []Window `yaml:"schedule"`
	DisplayOnly bool     `yaml:"display_only"`
	IPC         IPC      `yaml:"ipc"`
	Log         Log      `yaml:"log"`
}

// Default returns a Config populated with default values.
func Default() *Config {
	return &Config{
		Mode:        "always",
		DisplayOnly: false,
		IPC: IPC{
			PipeName: "keep-screen-awake",
			HTTPPort: 9877,
		},
		Log: Log{
			Level: "info",
			File:  "",
		},
	}
}

// Load reads a YAML config file from path. If path is empty or the file does
// not exist, Default() is returned without error.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("config: read %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse %q: %w", path, err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config: validate: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	switch cfg.Mode {
	case "always", "toggle", "schedule":
		// valid
	default:
		return fmt.Errorf("unknown mode %q; must be always, toggle, or schedule", cfg.Mode)
	}
	return nil
}
