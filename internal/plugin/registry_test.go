package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCloudProvider implements CloudProvider for testing.
type mockCloudProvider struct {
	name string
}

func (m *mockCloudProvider) Name() string                              { return m.name }
func (m *mockCloudProvider) Description() string                       { return "mock " + m.name }
func (m *mockCloudProvider) GenerateIaC(ctx *ProjectContext) error     { return nil }
func (m *mockCloudProvider) GenerateCIAuth(ctx *ProjectContext) error  { return nil }
func (m *mockCloudProvider) RegistryConfig() RegistryConfig            { return RegistryConfig{} }
func (m *mockCloudProvider) RequiredAPIs() []string                    { return nil }
func (m *mockCloudProvider) SupportedComponents() []string             { return nil }

func TestRegistry_CloudProvider(t *testing.T) {
	r := NewRegistry()

	// Register
	err := r.RegisterCloudProvider(&mockCloudProvider{name: "gcp"})
	require.NoError(t, err)

	// Get
	provider, err := r.GetCloudProvider("gcp")
	require.NoError(t, err)
	assert.Equal(t, "gcp", provider.Name())

	// Duplicate
	err = r.RegisterCloudProvider(&mockCloudProvider{name: "gcp"})
	require.Error(t, err)

	// Not found
	_, err = r.GetCloudProvider("aws")
	require.Error(t, err)

	// List
	names := r.ListCloudProviders()
	assert.Equal(t, []string{"gcp"}, names)
}

func TestRegistry_InfraComponent(t *testing.T) {
	r := NewRegistry()

	comp := &mockInfraComponent{name: "kafka"}
	err := r.RegisterInfraComponent(comp)
	require.NoError(t, err)

	got, err := r.GetInfraComponent("kafka")
	require.NoError(t, err)
	assert.Equal(t, "kafka", got.Name())

	names := r.ListInfraComponents()
	assert.Equal(t, []string{"kafka"}, names)
}

type mockInfraComponent struct {
	name string
}

func (m *mockInfraComponent) Name() string                              { return m.name }
func (m *mockInfraComponent) Description() string                       { return "mock " + m.name }
func (m *mockInfraComponent) Tier() int                                 { return 0 }
func (m *mockInfraComponent) DependsOn() []string                       { return nil }
func (m *mockInfraComponent) GenerateHelm(ctx *ProjectContext) error    { return nil }
func (m *mockInfraComponent) TiltSetup() string                         { return "" }
func (m *mockInfraComponent) DefaultConfig() map[string]interface{}     { return nil }
