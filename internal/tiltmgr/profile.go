// Package tiltmgr provides Tilt profile management, service discovery, and process control.
package tiltmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile represents a Tilt launch profile with selected services and infrastructure.
type Profile struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Services    []string `yaml:"services"`
	Infra       []string `yaml:"infra"`
}

// ProfileManager handles profile CRUD operations.
type ProfileManager struct {
	profilesDir string
}

// NewProfileManager creates a profile manager for the given profiles directory.
func NewProfileManager(profilesDir string) *ProfileManager {
	return &ProfileManager{profilesDir: profilesDir}
}

// EnsureDir creates the profiles directory if it doesn't exist.
func (pm *ProfileManager) EnsureDir() error {
	return os.MkdirAll(pm.profilesDir, 0o755)
}

// Load reads a profile by name.
func (pm *ProfileManager) Load(name string) (*Profile, error) {
	data, err := os.ReadFile(pm.profilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		return nil, fmt.Errorf("read profile %q: %w", name, err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", name, err)
	}
	return &p, nil
}

// Save writes a profile to disk.
func (pm *ProfileManager) Save(p *Profile) error {
	if err := pm.EnsureDir(); err != nil {
		return err
	}

	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}
	return os.WriteFile(pm.profilePath(p.Name), data, 0o644)
}

// Delete removes a profile by name.
func (pm *ProfileManager) Delete(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete the default profile")
	}
	path := pm.profilePath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", name)
	}
	return os.Remove(path)
}

// List returns all available profiles sorted by name.
func (pm *ProfileManager) List() ([]*Profile, error) {
	entries, err := os.ReadDir(pm.profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	var profiles []*Profile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		p, err := pm.Load(name)
		if err != nil {
			continue // Skip corrupt profiles
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		// Default profile always first
		if profiles[i].Name == "default" {
			return true
		}
		if profiles[j].Name == "default" {
			return false
		}
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// Exists checks if a profile exists.
func (pm *ProfileManager) Exists(name string) bool {
	_, err := os.Stat(pm.profilePath(name))
	return err == nil
}

func (pm *ProfileManager) profilePath(name string) string {
	return filepath.Join(pm.profilesDir, name+".yaml")
}
