package tiltmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTiltfile_Basic(t *testing.T) {
	dir := t.TempDir()
	tiltfile := filepath.Join(dir, "Tiltfile")

	content := `
load('./utils/tilt/node.tilt', 'create_node_service')
load('./k8s/helm/infra/postgresql/Tiltfile', 'postgresql_setup')
load('./k8s/helm/infra/redis/Tiltfile', 'redis_setup')

docker_build('registry/auth-service', '.')
docker_build('registry/api-service', '.')

k8s_resource('auth-service', labels=['auth'])
k8s_resource('api-service', labels=['api'])

local_resource('yarn-install', cmd='yarn install')
`
	err := os.WriteFile(tiltfile, []byte(content), 0o644)
	require.NoError(t, err)

	discovered, err := ParseTiltfile(tiltfile)
	require.NoError(t, err)

	// Check infra
	assert.Contains(t, discovered.Infra, "postgresql")
	assert.Contains(t, discovered.Infra, "redis")
	assert.Contains(t, discovered.Infra, "yarn-install")

	// Check services
	serviceNames := make([]string, 0, len(discovered.Services))
	for _, s := range discovered.Services {
		serviceNames = append(serviceNames, s.Name)
	}
	assert.Contains(t, serviceNames, "auth-service")
	assert.Contains(t, serviceNames, "api-service")

	// Services should NOT include infra
	assert.NotContains(t, serviceNames, "postgresql")
	assert.NotContains(t, serviceNames, "redis")
	assert.NotContains(t, serviceNames, "yarn-install")
}

func TestParseTiltfile_GroupInference(t *testing.T) {
	dir := t.TempDir()
	tiltfile := filepath.Join(dir, "Tiltfile")

	content := `
docker_build('registry/billing-service', './apps/billing-service')
k8s_resource('billing-service', labels=['00_billing'])
`
	err := os.WriteFile(tiltfile, []byte(content), 0o644)
	require.NoError(t, err)

	discovered, err := ParseTiltfile(tiltfile)
	require.NoError(t, err)

	require.Len(t, discovered.Services, 1)
	assert.Equal(t, "billing-service", discovered.Services[0].Name)
	assert.Equal(t, "billing", discovered.Services[0].Group) // Strips numeric prefix
}

func TestParseTiltfile_HelmPathGrouping(t *testing.T) {
	dir := t.TempDir()
	tiltfile := filepath.Join(dir, "Tiltfile")

	content := `
docker_build('registry/user-service', '.')
# Reference to k8s/helm/apps/identity/user-service
k8s_resource('user-service')
`
	// Add helm path reference
	content = `
load('./k8s/helm/apps/identity/user-service/Tiltfile', 'setup')
k8s_resource('user-service')
`
	err := os.WriteFile(tiltfile, []byte(content), 0o644)
	require.NoError(t, err)

	// Create the sub-Tiltfile
	subDir := filepath.Join(dir, "k8s/helm/apps/identity/user-service")
	err = os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	subTiltfile := filepath.Join(subDir, "Tiltfile")
	err = os.WriteFile(subTiltfile, []byte(`k8s_resource('user-service')`), 0o644)
	require.NoError(t, err)

	discovered, err := ParseTiltfile(tiltfile)
	require.NoError(t, err)

	require.Len(t, discovered.Services, 1)
	assert.Equal(t, "user-service", discovered.Services[0].Name)
}

func TestParseTiltfile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	tiltfile := filepath.Join(dir, "Tiltfile")
	err := os.WriteFile(tiltfile, []byte(""), 0o644)
	require.NoError(t, err)

	discovered, err := ParseTiltfile(tiltfile)
	require.NoError(t, err)
	assert.Empty(t, discovered.Services)
	assert.Empty(t, discovered.Infra)
}

func TestParseTiltfile_NotFound(t *testing.T) {
	_, err := ParseTiltfile("/nonexistent/Tiltfile")
	assert.Error(t, err)
}

func TestInferGroup_NumericPrefix(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"numeric prefix", `k8s_resource('svc', labels=['01_auth'])`, "auth"},
		{"no prefix", `k8s_resource('svc', labels=['billing'])`, "billing"},
		{"default", `k8s_resource('svc')`, "default"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := inferGroup("svc", tc.content)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNumeric(t *testing.T) {
	assert.True(t, isNumeric("00"))
	assert.True(t, isNumeric("123"))
	assert.False(t, isNumeric("abc"))
	assert.False(t, isNumeric(""))
	assert.False(t, isNumeric("12a"))
}
