package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a .workstats.yml file and merges it over Defaults().
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(data)
}

// Parse unmarshals YAML bytes over Defaults().
func Parse(data []byte) (Config, error) {
	cfg := Defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Workspace.MaxDepth <= 0 {
		cfg.Workspace.MaxDepth = DefaultMaxDepth
	}
	if cfg.Types == nil {
		cfg.Types = DefaultTypeWeights()
	} else {
		// Fill missing type keys from defaults without overriding explicit values.
		for k, v := range DefaultTypeWeights() {
			if _, ok := cfg.Types[k]; !ok {
				cfg.Types[k] = v
			}
		}
	}
	if cfg.Analysis.ChangeUnit == "" {
		cfg.Analysis.ChangeUnit = "commit"
	}
	return cfg, nil
}
