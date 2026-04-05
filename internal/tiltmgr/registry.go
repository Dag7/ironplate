package tiltmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Registry represents the tilt/registry.yaml data model.
type Registry struct {
	Services       map[string]ServiceEntry `yaml:"services"`
	Infrastructure map[string]InfraEntry   `yaml:"infrastructure"`
}

// ServiceEntry represents a service in the registry.
type ServiceEntry struct {
	Type      string   `yaml:"type"`
	Group     string   `yaml:"group"`
	Port      int      `yaml:"port"`
	DebugPort int      `yaml:"debugPort"`
	Src       string   `yaml:"src"`
	Labels    []string `yaml:"labels"`
}

// InfraEntry represents an infrastructure component in the registry.
type InfraEntry struct {
	Enabled  bool     `yaml:"enabled"`
	Local    bool     `yaml:"local"`
	Required bool     `yaml:"required"`
	Deps     []string `yaml:"deps"`
	HelmPath string   `yaml:"helmPath"`
	SetupFn  string   `yaml:"setupFn"`
}

// DiscoveredService represents a service found in the registry.
type DiscoveredService struct {
	Name      string
	Group     string
	Type      string
	Port      int
	DebugPort int
	Labels    []string
}

// DiscoveredResources contains all resources from the registry.
type DiscoveredResources struct {
	Services []DiscoveredService
	Infra    []DiscoveredInfra
}

// DiscoveredInfra represents an infrastructure component from the registry.
type DiscoveredInfra struct {
	Name     string
	Enabled  bool
	Local    bool
	Required bool
	Deps     []string
}

// LoadRegistry reads the tilt/registry.yaml file.
func LoadRegistry(projectRoot string) (*Registry, error) {
	regPath := filepath.Join(projectRoot, "tilt", "registry.yaml")
	data, err := os.ReadFile(regPath)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &reg, nil
}

// Discover returns structured resources from the registry.
func Discover(projectRoot string) (*DiscoveredResources, error) {
	reg, err := LoadRegistry(projectRoot)
	if err != nil {
		return nil, err
	}

	var services []DiscoveredService
	for name, entry := range reg.Services {
		services = append(services, DiscoveredService{
			Name:      name,
			Group:     entry.Group,
			Type:      entry.Type,
			Port:      entry.Port,
			DebugPort: entry.DebugPort,
			Labels:    entry.Labels,
		})
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].Group != services[j].Group {
			return services[i].Group < services[j].Group
		}
		return services[i].Name < services[j].Name
	})

	var infra []DiscoveredInfra
	for name, entry := range reg.Infrastructure {
		infra = append(infra, DiscoveredInfra{
			Name:     name,
			Enabled:  entry.Enabled,
			Local:    entry.Local,
			Required: entry.Required,
			Deps:     entry.Deps,
		})
	}
	sort.Slice(infra, func(i, j int) bool {
		return infra[i].Name < infra[j].Name
	})

	return &DiscoveredResources{
		Services: services,
		Infra:    infra,
	}, nil
}
