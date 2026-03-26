package tiltmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileManager_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	profile := &Profile{
		Name:        "test",
		Description: "Test profile",
		Services:    []string{"auth-service", "api-service"},
		Infra:       []string{"postgresql", "redis"},
	}

	err := pm.Save(profile)
	require.NoError(t, err)

	loaded, err := pm.Load("test")
	require.NoError(t, err)

	assert.Equal(t, profile.Name, loaded.Name)
	assert.Equal(t, profile.Description, loaded.Description)
	assert.Equal(t, profile.Services, loaded.Services)
	assert.Equal(t, profile.Infra, loaded.Infra)
}

func TestProfileManager_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	_, err := pm.Load("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProfileManager_List(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	// Save multiple profiles
	for _, name := range []string{"backend", "frontend", "default"} {
		err := pm.Save(&Profile{
			Name:     name,
			Services: []string{"svc-" + name},
		})
		require.NoError(t, err)
	}

	profiles, err := pm.List()
	require.NoError(t, err)
	require.Len(t, profiles, 3)

	// Default should be first
	assert.Equal(t, "default", profiles[0].Name)
	// Rest alphabetical
	assert.Equal(t, "backend", profiles[1].Name)
	assert.Equal(t, "frontend", profiles[2].Name)
}

func TestProfileManager_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	profiles, err := pm.List()
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

func TestProfileManager_ListNonexistentDir(t *testing.T) {
	pm := NewProfileManager("/nonexistent/path")

	profiles, err := pm.List()
	require.NoError(t, err)
	assert.Nil(t, profiles)
}

func TestProfileManager_Delete(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	err := pm.Save(&Profile{Name: "deleteme"})
	require.NoError(t, err)
	assert.True(t, pm.Exists("deleteme"))

	err = pm.Delete("deleteme")
	require.NoError(t, err)
	assert.False(t, pm.Exists("deleteme"))
}

func TestProfileManager_DeleteDefault(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	err := pm.Delete("default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete")
}

func TestProfileManager_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	err := pm.Delete("nonexistent")
	assert.Error(t, err)
}

func TestProfileManager_Exists(t *testing.T) {
	dir := t.TempDir()
	pm := NewProfileManager(dir)

	assert.False(t, pm.Exists("nope"))

	err := pm.Save(&Profile{Name: "yes"})
	require.NoError(t, err)
	assert.True(t, pm.Exists("yes"))
}

func TestProfileManager_EnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "profiles")
	pm := NewProfileManager(dir)

	err := pm.EnsureDir()
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
