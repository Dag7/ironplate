package tiltmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRegistryYAML(t *testing.T, dir, content string) {
	t.Helper()
	tiltDir := filepath.Join(dir, "tilt")
	require.NoError(t, os.MkdirAll(tiltDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tiltDir, "registry.yaml"), []byte(content), 0o644))
}

func TestLoadRegistry_Basic(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `
services:
  auth-service:
    type: node-api
    group: auth
    port: 3001
    debugPort: 9221
    src: apps/auth-service
    labels: [auth, core]
  api-service:
    type: go-api
    group: core
    port: 3002
    src: apps/api-service
infrastructure:
  postgresql:
    enabled: true
    local: true
    required: true
  redis:
    enabled: true
    local: true
    deps: []
`)

	reg, err := LoadRegistry(dir)
	require.NoError(t, err)

	assert.Len(t, reg.Services, 2)
	assert.Equal(t, "node-api", reg.Services["auth-service"].Type)
	assert.Equal(t, "auth", reg.Services["auth-service"].Group)
	assert.Equal(t, 3001, reg.Services["auth-service"].Port)
	assert.Equal(t, 9221, reg.Services["auth-service"].DebugPort)
	assert.Equal(t, []string{"auth", "core"}, reg.Services["auth-service"].Labels)

	assert.Len(t, reg.Infrastructure, 2)
	assert.True(t, reg.Infrastructure["postgresql"].Required)
	assert.True(t, reg.Infrastructure["postgresql"].Local)
	assert.True(t, reg.Infrastructure["redis"].Enabled)
}

func TestLoadRegistry_NotFound(t *testing.T) {
	_, err := LoadRegistry("/nonexistent")
	assert.Error(t, err)
}

func TestLoadRegistry_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `{{{invalid yaml`)

	_, err := LoadRegistry(dir)
	assert.Error(t, err)
}

func TestDiscover_Basic(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `
services:
  billing-service:
    type: node-api
    group: billing
    port: 3010
    labels: [billing]
  web:
    type: nextjs
    group: frontend
    port: 3000
infrastructure:
  postgresql:
    enabled: true
    required: true
    local: true
  kafka:
    enabled: true
    deps: [postgresql]
  redis:
    enabled: true
    local: true
`)

	discovered, err := Discover(dir)
	require.NoError(t, err)

	// Services sorted by group then name
	require.Len(t, discovered.Services, 2)
	assert.Equal(t, "billing-service", discovered.Services[0].Name)
	assert.Equal(t, "billing", discovered.Services[0].Group)
	assert.Equal(t, 3010, discovered.Services[0].Port)
	assert.Equal(t, "web", discovered.Services[1].Name)
	assert.Equal(t, "frontend", discovered.Services[1].Group)

	// Infra sorted by name
	require.Len(t, discovered.Infra, 3)
	assert.Equal(t, "kafka", discovered.Infra[0].Name)
	assert.Equal(t, []string{"postgresql"}, discovered.Infra[0].Deps)
	assert.Equal(t, "postgresql", discovered.Infra[1].Name)
	assert.True(t, discovered.Infra[1].Required)
	assert.Equal(t, "redis", discovered.Infra[2].Name)
	assert.True(t, discovered.Infra[2].Local)
}

func TestDiscover_EmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `
services: {}
infrastructure: {}
`)

	discovered, err := Discover(dir)
	require.NoError(t, err)
	assert.Empty(t, discovered.Services)
	assert.Empty(t, discovered.Infra)
}

func TestDiscover_ServicesOnly(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `
services:
  api:
    type: node-api
    group: core
    port: 3001
`)

	discovered, err := Discover(dir)
	require.NoError(t, err)
	assert.Len(t, discovered.Services, 1)
	assert.Empty(t, discovered.Infra)
}

func TestDiscover_InfraOnly(t *testing.T) {
	dir := t.TempDir()
	writeRegistryYAML(t, dir, `
infrastructure:
  postgresql:
    enabled: true
    required: true
`)

	discovered, err := Discover(dir)
	require.NoError(t, err)
	assert.Empty(t, discovered.Services)
	assert.Len(t, discovered.Infra, 1)
	assert.Equal(t, "postgresql", discovered.Infra[0].Name)
}
