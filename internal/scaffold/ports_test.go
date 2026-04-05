package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dag7/ironplate/internal/config"
)

func TestNextForwardPort_Empty(t *testing.T) {
	port := NextForwardPort(nil)
	assert.Equal(t, BaseForwardPort, port)
}

func TestNextForwardPort_WithExisting(t *testing.T) {
	existing := []config.ServiceSpec{
		{Name: "api", Port: 3010},
		{Name: "web", Port: 3011},
	}
	port := NextForwardPort(existing)
	assert.Equal(t, 3012, port)
}

func TestNextForwardPort_NonSequential(t *testing.T) {
	existing := []config.ServiceSpec{
		{Name: "api", Port: 3010},
		{Name: "web", Port: 3050},
	}
	port := NextForwardPort(existing)
	assert.Equal(t, 3051, port)
}

func TestNextDebugForwardPort_NodeEmpty(t *testing.T) {
	port := NextDebugForwardPort(nil, "node-api")
	assert.Equal(t, BaseNodeDebugPort, port)
}

func TestNextDebugForwardPort_GoEmpty(t *testing.T) {
	port := NextDebugForwardPort(nil, "go-api")
	assert.Equal(t, BaseGoDebugPort, port)
}

func TestNextDebugForwardPort_GoWithExisting(t *testing.T) {
	existing := []config.ServiceSpec{
		{Name: "api-go", Type: "go-api", Port: 3010},
	}
	port := NextDebugForwardPort(existing, "go-api")
	assert.Equal(t, BaseGoDebugPort+1, port)
}

func TestPortAllocator_NodeOnly(t *testing.T) {
	pa := newPortAllocator()

	api := pa.allocate("api", "node-api", "core")
	assert.Equal(t, BaseForwardPort, api.Port)
	assert.Equal(t, BaseNodeDebugPort, api.DebugPort)

	web := pa.allocate("web", "nextjs", "frontend")
	assert.Equal(t, BaseForwardPort+1, web.Port)
	assert.Equal(t, BaseNodeDebugPort+1, web.DebugPort)
}

func TestPortAllocator_GoOnly(t *testing.T) {
	pa := newPortAllocator()

	api := pa.allocate("api", "go-api", "core")
	assert.Equal(t, BaseForwardPort, api.Port)
	assert.Equal(t, BaseGoDebugPort, api.DebugPort)
}

func TestPortAllocator_Mixed(t *testing.T) {
	pa := newPortAllocator()

	api := pa.allocate("api", "node-api", "core")
	web := pa.allocate("web", "nextjs", "frontend")
	goApi := pa.allocate("api-go", "go-api", "core")

	// HTTP ports increment sequentially
	assert.Equal(t, BaseForwardPort, api.Port)
	assert.Equal(t, BaseForwardPort+1, web.Port)
	assert.Equal(t, BaseForwardPort+2, goApi.Port)

	// Debug ports use separate counters per language
	assert.Equal(t, BaseNodeDebugPort, api.DebugPort)
	assert.Equal(t, BaseNodeDebugPort+1, web.DebugPort)
	assert.Equal(t, BaseGoDebugPort, goApi.DebugPort)
}

func TestDefaultExampleServices_NodeOnly(t *testing.T) {
	cfg := &config.ProjectConfig{
		Spec: config.ProjectSpec{Languages: []string{"node"}},
	}

	services := DefaultExampleServices(cfg)

	assert.Len(t, services, 2)
	assert.Equal(t, "api", services[0].Name)
	assert.Equal(t, "node-api", services[0].Type)
	assert.Equal(t, BaseForwardPort, services[0].Port)
	assert.Equal(t, BaseNodeDebugPort, services[0].DebugPort)

	assert.Equal(t, "web", services[1].Name)
	assert.Equal(t, "nextjs", services[1].Type)
	assert.Equal(t, BaseForwardPort+1, services[1].Port)
	assert.Equal(t, BaseNodeDebugPort+1, services[1].DebugPort)
}

func TestDefaultExampleServices_GoOnly(t *testing.T) {
	cfg := &config.ProjectConfig{
		Spec: config.ProjectSpec{Languages: []string{"go"}},
	}

	services := DefaultExampleServices(cfg)

	assert.Len(t, services, 1)
	assert.Equal(t, "api", services[0].Name)
	assert.Equal(t, "go-api", services[0].Type)
	assert.Equal(t, BaseForwardPort, services[0].Port)
	assert.Equal(t, BaseGoDebugPort, services[0].DebugPort)
}

func TestDefaultExampleServices_Mixed(t *testing.T) {
	cfg := &config.ProjectConfig{
		Spec: config.ProjectSpec{Languages: []string{"node", "go"}},
	}

	services := DefaultExampleServices(cfg)

	assert.Len(t, services, 3)
	assert.Equal(t, "api", services[0].Name)
	assert.Equal(t, "web", services[1].Name)
	assert.Equal(t, "api-go", services[2].Name)

	// All ports are unique
	ports := make(map[int]bool)
	for _, svc := range services {
		assert.False(t, ports[svc.Port], "duplicate port %d for %s", svc.Port, svc.Name)
		ports[svc.Port] = true
	}
}

func TestContainerPortConstants(t *testing.T) {
	assert.Equal(t, 3000, ContainerHTTPPort)
	assert.Equal(t, 9229, ContainerNodeDebugPort)
	assert.Equal(t, 40000, ContainerGoDebugPort)
}
