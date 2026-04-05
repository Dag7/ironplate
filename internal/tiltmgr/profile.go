// Package tiltmgr provides Tilt profile management, service discovery, and process control.
package tiltmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ProfilesConfig represents the tilt/profiles.yaml data model.
type ProfilesConfig struct {
	Active   string                    `yaml:"active"`
	Profiles map[string]ProfileDef     `yaml:"profiles"`
}

// ProfileDef represents a profile definition from profiles.yaml.
// Services and Infra are interface{} because they can be:
//   - string: "all" or "auto" (infra only)
//   - []interface{}: explicit list of names
//   - map[string]interface{}: filter with groups/labels
type ProfileDef struct {
	Description string      `yaml:"description"`
	Services    interface{} `yaml:"services"`
	Infra       interface{} `yaml:"infra"`
}

// Profile represents a resolved profile ready for display or use.
type Profile struct {
	Name        string
	Description string
	ServicesRaw interface{} // raw value from YAML for display
	InfraRaw    interface{} // raw value from YAML for display
}

// ProfileManager handles profile operations backed by tilt/profiles.yaml.
type ProfileManager struct {
	projectRoot string
}

// NewProfileManager creates a profile manager for the given project root.
func NewProfileManager(projectRoot string) *ProfileManager {
	return &ProfileManager{projectRoot: projectRoot}
}

func (pm *ProfileManager) profilesPath() string {
	return filepath.Join(pm.projectRoot, "tilt", "profiles.yaml")
}

func (pm *ProfileManager) loadConfig() (*ProfilesConfig, error) {
	data, err := os.ReadFile(pm.profilesPath())
	if err != nil {
		return nil, fmt.Errorf("read profiles.yaml: %w", err)
	}
	var cfg ProfilesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse profiles.yaml: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]ProfileDef)
	}
	return &cfg, nil
}

func (pm *ProfileManager) saveConfig(cfg *ProfilesConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal profiles.yaml: %w", err)
	}
	return os.WriteFile(pm.profilesPath(), data, 0o644)
}

// ActiveProfile returns the name of the active profile.
func (pm *ProfileManager) ActiveProfile() (string, error) {
	cfg, err := pm.loadConfig()
	if err != nil {
		return "", err
	}
	if cfg.Active == "" {
		return "full", nil
	}
	return cfg.Active, nil
}

// SetActive updates the active profile in profiles.yaml.
func (pm *ProfileManager) SetActive(name string) error {
	cfg, err := pm.loadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	cfg.Active = name
	return pm.saveConfig(cfg)
}

// List returns all profile names with metadata, sorted alphabetically.
func (pm *ProfileManager) List() ([]*Profile, error) {
	cfg, err := pm.loadConfig()
	if err != nil {
		return nil, err
	}

	var profiles []*Profile
	for name, def := range cfg.Profiles {
		profiles = append(profiles, &Profile{
			Name:        name,
			Description: def.Description,
			ServicesRaw: def.Services,
			InfraRaw:    def.Infra,
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// Load reads a single profile by name.
func (pm *ProfileManager) Load(name string) (*Profile, error) {
	cfg, err := pm.loadConfig()
	if err != nil {
		return nil, err
	}
	def, ok := cfg.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found (available: %s)", name, pm.availableNames(cfg))
	}
	return &Profile{
		Name:        name,
		Description: def.Description,
		ServicesRaw: def.Services,
		InfraRaw:    def.Infra,
	}, nil
}

// Exists checks if a profile exists.
func (pm *ProfileManager) Exists(name string) bool {
	cfg, err := pm.loadConfig()
	if err != nil {
		return false
	}
	_, ok := cfg.Profiles[name]
	return ok
}

// Save writes or updates a profile definition in profiles.yaml.
func (pm *ProfileManager) Save(name, description string, services, infra interface{}) error {
	cfg, err := pm.loadConfig()
	if err != nil {
		return err
	}
	cfg.Profiles[name] = ProfileDef{
		Description: description,
		Services:    services,
		Infra:       infra,
	}
	return pm.saveConfig(cfg)
}

// Delete removes a profile from profiles.yaml.
func (pm *ProfileManager) Delete(name string) error {
	cfg, err := pm.loadConfig()
	if err != nil {
		return err
	}
	builtins := map[string]bool{"minimal": true, "core": true, "full": true, "infra-only": true}
	if builtins[name] {
		return fmt.Errorf("cannot delete built-in profile %q", name)
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(cfg.Profiles, name)
	if cfg.Active == name {
		cfg.Active = "full"
	}
	return pm.saveConfig(cfg)
}

func (pm *ProfileManager) availableNames(cfg *ProfilesConfig) string {
	var names []string
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}

// FormatServicesDisplay returns a human-readable description of the services config.
func FormatServicesDisplay(raw interface{}) string {
	if raw == nil {
		return "none"
	}
	switch v := raw.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) == 0 {
			return "none"
		}
		names := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		return fmt.Sprintf("%d (%s)", len(names), joinMax(names, 4))
	case map[string]interface{}:
		parts := []string{}
		if groups, ok := v["groups"]; ok {
			if gl, ok := groups.([]interface{}); ok {
				gs := make([]string, 0, len(gl))
				for _, g := range gl {
					if s, ok := g.(string); ok {
						gs = append(gs, s)
					}
				}
				parts = append(parts, fmt.Sprintf("groups: %v", gs))
			}
		}
		if labels, ok := v["labels"]; ok {
			if ll, ok := labels.([]interface{}); ok {
				ls := make([]string, 0, len(ll))
				for _, l := range ll {
					if s, ok := l.(string); ok {
						ls = append(ls, s)
					}
				}
				parts = append(parts, fmt.Sprintf("labels: %v", ls))
			}
		}
		if len(parts) == 0 {
			return "filter"
		}
		result := ""
		for i, p := range parts {
			if i > 0 {
				result += ", "
			}
			result += p
		}
		return result
	}
	return fmt.Sprintf("%v", raw)
}

// FormatInfraDisplay returns a human-readable description of the infra config.
func FormatInfraDisplay(raw interface{}) string {
	if raw == nil {
		return "none"
	}
	switch v := raw.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) == 0 {
			return "none"
		}
		names := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		return fmt.Sprintf("%d (%s)", len(names), joinMax(names, 4))
	}
	return fmt.Sprintf("%v", raw)
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		result := ""
		for i, s := range items {
			if i > 0 {
				result += ", "
			}
			result += s
		}
		return result
	}
	result := ""
	for i := 0; i < max; i++ {
		if i > 0 {
			result += ", "
		}
		result += items[i]
	}
	return result + fmt.Sprintf(", +%d more", len(items)-max)
}
