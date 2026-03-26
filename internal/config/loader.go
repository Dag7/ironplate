package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFile = "ironplate.yaml"
	CurrentAPIVersion = "ironplate.dev/v1"
	CurrentKind       = "Project"
)

// Load reads and parses an ironplate.yaml configuration file.
func Load(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	return Parse(data)
}

// Parse parses YAML data into a ProjectConfig.
func Parse(data []byte) (*ProjectConfig, error) {
	cfg := &ProjectConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

// FindConfigFile searches for ironplate.yaml starting from the given directory.
func FindConfigFile(startDir string) (string, error) {
	dir := startDir
	for {
		path := filepath.Join(dir, DefaultConfigFile)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no %s found in %s or any parent directory", DefaultConfigFile, startDir)
}

// validate performs structural validation on a ProjectConfig.
func validate(cfg *ProjectConfig) error {
	if cfg.APIVersion != CurrentAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q, expected %q", cfg.APIVersion, CurrentAPIVersion)
	}
	if cfg.Kind != CurrentKind {
		return fmt.Errorf("unsupported kind %q, expected %q", cfg.Kind, CurrentKind)
	}
	if cfg.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if cfg.Metadata.Organization == "" {
		return fmt.Errorf("metadata.organization is required")
	}
	if len(cfg.Spec.Languages) == 0 {
		return fmt.Errorf("spec.languages must have at least one entry")
	}

	for _, lang := range cfg.Spec.Languages {
		if lang != "node" && lang != "go" {
			return fmt.Errorf("unsupported language %q, must be 'node' or 'go'", lang)
		}
	}

	if p := cfg.Spec.Cloud.Provider; p != "" && p != "gcp" && p != "aws" && p != "azure" && p != "none" {
		return fmt.Errorf("unsupported cloud provider %q", p)
	}

	return nil
}
