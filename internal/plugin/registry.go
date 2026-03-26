package plugin

import (
	"fmt"
	"sync"
)

// Registry holds all registered plugins in thread-safe maps.
// Uses the Registry pattern for extensible plugin management.
type Registry struct {
	mu              sync.RWMutex
	cloudProviders  map[string]CloudProvider
	serviceGens     map[string]ServiceGenerator
	packageGens     map[string]PackageGenerator
	infraComponents map[string]InfraComponent
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		cloudProviders:  make(map[string]CloudProvider),
		serviceGens:     make(map[string]ServiceGenerator),
		packageGens:     make(map[string]PackageGenerator),
		infraComponents: make(map[string]InfraComponent),
	}
}

// RegisterCloudProvider registers a cloud provider plugin.
func (r *Registry) RegisterCloudProvider(p CloudProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.cloudProviders[p.Name()]; exists {
		return fmt.Errorf("cloud provider %q already registered", p.Name())
	}
	r.cloudProviders[p.Name()] = p
	return nil
}

// GetCloudProvider returns a registered cloud provider by name.
func (r *Registry) GetCloudProvider(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.cloudProviders[name]
	if !ok {
		return nil, fmt.Errorf("cloud provider %q not found", name)
	}
	return p, nil
}

// ListCloudProviders returns all registered cloud provider names.
func (r *Registry) ListCloudProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.cloudProviders))
	for name := range r.cloudProviders {
		names = append(names, name)
	}
	return names
}

// RegisterServiceGenerator registers a service generator plugin.
func (r *Registry) RegisterServiceGenerator(g ServiceGenerator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.serviceGens[g.Name()]; exists {
		return fmt.Errorf("service generator %q already registered", g.Name())
	}
	r.serviceGens[g.Name()] = g
	return nil
}

// GetServiceGenerator returns a registered service generator by name.
func (r *Registry) GetServiceGenerator(name string) (ServiceGenerator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.serviceGens[name]
	if !ok {
		return nil, fmt.Errorf("service generator %q not found", name)
	}
	return g, nil
}

// ListServiceGenerators returns all registered service generator names.
func (r *Registry) ListServiceGenerators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.serviceGens))
	for name := range r.serviceGens {
		names = append(names, name)
	}
	return names
}

// RegisterPackageGenerator registers a package generator plugin.
func (r *Registry) RegisterPackageGenerator(g PackageGenerator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.packageGens[g.Name()]; exists {
		return fmt.Errorf("package generator %q already registered", g.Name())
	}
	r.packageGens[g.Name()] = g
	return nil
}

// GetPackageGenerator returns a registered package generator by name.
func (r *Registry) GetPackageGenerator(name string) (PackageGenerator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	g, ok := r.packageGens[name]
	if !ok {
		return nil, fmt.Errorf("package generator %q not found", name)
	}
	return g, nil
}

// RegisterInfraComponent registers an infrastructure component plugin.
func (r *Registry) RegisterInfraComponent(c InfraComponent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.infraComponents[c.Name()]; exists {
		return fmt.Errorf("infra component %q already registered", c.Name())
	}
	r.infraComponents[c.Name()] = c
	return nil
}

// GetInfraComponent returns a registered infrastructure component by name.
func (r *Registry) GetInfraComponent(name string) (InfraComponent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.infraComponents[name]
	if !ok {
		return nil, fmt.Errorf("infra component %q not found", name)
	}
	return c, nil
}

// ListInfraComponents returns all registered infrastructure component names.
func (r *Registry) ListInfraComponents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.infraComponents))
	for name := range r.infraComponents {
		names = append(names, name)
	}
	return names
}
