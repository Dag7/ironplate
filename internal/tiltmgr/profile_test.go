package tiltmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeProfilesYAML(t *testing.T, dir, content string) {
	t.Helper()
	tiltDir := filepath.Join(dir, "tilt")
	require.NoError(t, os.MkdirAll(tiltDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tiltDir, "profiles.yaml"), []byte(content), 0o644))
}

func TestProfileManager_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    description: "All services"
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	err := pm.Save("test", "Test profile", []string{"auth-service", "api-service"}, []string{"postgresql", "redis"})
	require.NoError(t, err)

	loaded, err := pm.Load("test")
	require.NoError(t, err)

	assert.Equal(t, "test", loaded.Name)
	assert.Equal(t, "Test profile", loaded.Description)
}

func TestProfileManager_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    description: "All services"
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	_, err := pm.Load("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfileManager_List(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    description: "Everything"
    services: "all"
    infra: "all"
  minimal:
    description: "Bare minimum"
    services: []
    infra: [postgresql]
  core:
    description: "Core services"
    services:
      groups: [core]
    infra: "auto"
`)
	pm := NewProfileManager(dir)

	profiles, err := pm.List()
	require.NoError(t, err)
	require.Len(t, profiles, 3)

	// Sorted alphabetically
	assert.Equal(t, "core", profiles[0].Name)
	assert.Equal(t, "full", profiles[1].Name)
	assert.Equal(t, "minimal", profiles[2].Name)
}

func TestProfileManager_ListNoFile(t *testing.T) {
	pm := NewProfileManager("/nonexistent/path")

	_, err := pm.List()
	assert.Error(t, err)
}

func TestProfileManager_ActiveProfile(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: minimal
profiles:
  full:
    services: "all"
    infra: "all"
  minimal:
    services: []
    infra: [postgresql]
`)
	pm := NewProfileManager(dir)

	active, err := pm.ActiveProfile()
	require.NoError(t, err)
	assert.Equal(t, "minimal", active)
}

func TestProfileManager_ActiveProfileDefault(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
profiles:
  full:
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	active, err := pm.ActiveProfile()
	require.NoError(t, err)
	assert.Equal(t, "full", active) // defaults to "full" when empty
}

func TestProfileManager_SetActive(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
  minimal:
    services: []
    infra: [postgresql]
`)
	pm := NewProfileManager(dir)

	err := pm.SetActive("minimal")
	require.NoError(t, err)

	active, err := pm.ActiveProfile()
	require.NoError(t, err)
	assert.Equal(t, "minimal", active)
}

func TestProfileManager_SetActiveNotFound(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	err := pm.SetActive("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfileManager_Delete(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
  deleteme:
    description: "To be deleted"
    services: []
    infra: []
`)
	pm := NewProfileManager(dir)

	assert.True(t, pm.Exists("deleteme"))

	err := pm.Delete("deleteme")
	require.NoError(t, err)
	assert.False(t, pm.Exists("deleteme"))
}

func TestProfileManager_DeleteBuiltin(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
  minimal:
    services: []
    infra: [postgresql]
`)
	pm := NewProfileManager(dir)

	err := pm.Delete("full")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete built-in")
}

func TestProfileManager_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	err := pm.Delete("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfileManager_DeleteActiveResetsToFull(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: custom
profiles:
  full:
    services: "all"
    infra: "all"
  custom:
    description: "Custom profile"
    services: []
    infra: []
`)
	pm := NewProfileManager(dir)

	err := pm.Delete("custom")
	require.NoError(t, err)

	active, err := pm.ActiveProfile()
	require.NoError(t, err)
	assert.Equal(t, "full", active)
}

func TestProfileManager_Exists(t *testing.T) {
	dir := t.TempDir()
	writeProfilesYAML(t, dir, `
active: full
profiles:
  full:
    services: "all"
    infra: "all"
`)
	pm := NewProfileManager(dir)

	assert.True(t, pm.Exists("full"))
	assert.False(t, pm.Exists("nope"))
}

func TestFormatServicesDisplay(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "none"},
		{"string all", "all", "all"},
		{"empty list", []interface{}{}, "none"},
		{"name list", []interface{}{"auth", "api", "web"}, "3 (auth, api, web)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, FormatServicesDisplay(tc.input))
		})
	}
}

func TestFormatInfraDisplay(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "none"},
		{"string all", "all", "all"},
		{"string auto", "auto", "auto"},
		{"empty list", []interface{}{}, "none"},
		{"name list", []interface{}{"postgresql", "redis"}, "2 (postgresql, redis)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, FormatInfraDisplay(tc.input))
		})
	}
}
