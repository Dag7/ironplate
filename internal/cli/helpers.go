package cli

import (
	"fmt"
	"path/filepath"

	"github.com/dag7/ironplate/internal/config"
)

// projectContext holds the resolved project config and root path.
// Used by CLI commands that operate on an existing ironplate project.
type projectContext struct {
	Config      *config.ProjectConfig
	ConfigPath  string
	ProjectRoot string
}

// loadProject finds and loads the ironplate project config from the current directory.
func loadProject() (*projectContext, error) {
	cfgPath, err := config.FindConfigFile(".")
	if err != nil {
		return nil, fmt.Errorf("not in an ironplate project: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return &projectContext{
		Config:      cfg,
		ConfigPath:  cfgPath,
		ProjectRoot: filepath.Dir(cfgPath),
	}, nil
}

// saveConfig marshals the project config and writes it to the config file.
func (pc *projectContext) saveConfig() error {
	return config.Save(pc.Config, pc.ConfigPath)
}
