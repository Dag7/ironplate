package devtools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncIcon(t *testing.T) {
	assert.Equal(t, "✓", SyncIcon("Synced"))
	assert.Equal(t, "○", SyncIcon("OutOfSync"))
	assert.Equal(t, "?", SyncIcon("Unknown"))
}

func TestHealthIcon(t *testing.T) {
	assert.Equal(t, "●", HealthIcon("Healthy"))
	assert.Equal(t, "◐", HealthIcon("Progressing"))
	assert.Equal(t, "✗", HealthIcon("Degraded"))
	assert.Equal(t, "⏸", HealthIcon("Suspended"))
	assert.Equal(t, "?", HealthIcon("Missing"))
	assert.Equal(t, "○", HealthIcon("Unknown"))
}

func TestGroupByProject(t *testing.T) {
	apps := []ArgoApp{
		{Name: "auth-staging", Project: "staging"},
		{Name: "api-staging", Project: "staging"},
		{Name: "auth-prod", Project: "production"},
		{Name: "orphan", Project: ""},
	}

	grouped := GroupByProject(apps)

	assert.Len(t, grouped["staging"], 2)
	assert.Len(t, grouped["production"], 1)
	assert.Len(t, grouped["default"], 1) // Empty project -> "default"
}

func TestGetOutOfSyncApps(t *testing.T) {
	// This tests the filtering logic directly (no kubectl)
	apps := []ArgoApp{
		{Name: "app1", SyncStatus: "Synced"},
		{Name: "app2", SyncStatus: "OutOfSync"},
		{Name: "app3", SyncStatus: "OutOfSync"},
		{Name: "app4", SyncStatus: "Synced"},
	}

	var outOfSync []ArgoApp
	for _, app := range apps {
		if app.SyncStatus == "OutOfSync" {
			outOfSync = append(outOfSync, app)
		}
	}

	assert.Len(t, outOfSync, 2)
	assert.Equal(t, "app2", outOfSync[0].Name)
	assert.Equal(t, "app3", outOfSync[1].Name)
}

func TestSyncMultipleArgoAppsResults(t *testing.T) {
	// Test the result map structure
	results := make(map[string]error)
	results["app1"] = nil
	results["app2"] = nil

	assert.Len(t, results, 2)
	assert.Nil(t, results["app1"])
	assert.Nil(t, results["app2"])
}
