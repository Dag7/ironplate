package engine

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderString_Simple(t *testing.T) {
	r := NewRenderer()

	ctx := map[string]string{"Name": "ironplate"}
	result, err := r.RenderString("Hello, {{ .Name }}!", ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, ironplate!", result)
}

func TestRenderString_WithFunctions(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		name     string
		tmpl     string
		ctx      interface{}
		expected string
	}{
		{
			name:     "kebabCase",
			tmpl:     `{{ kebabCase .Name }}`,
			ctx:      map[string]string{"Name": "MyProject"},
			expected: "my-project",
		},
		{
			name:     "snakeCase",
			tmpl:     `{{ snakeCase .Name }}`,
			ctx:      map[string]string{"Name": "MyProject"},
			expected: "my_project",
		},
		{
			name:     "camelCase",
			tmpl:     `{{ camelCase .Name }}`,
			ctx:      map[string]string{"Name": "my-project"},
			expected: "myProject",
		},
		{
			name:     "pascalCase",
			tmpl:     `{{ pascalCase .Name }}`,
			ctx:      map[string]string{"Name": "my-project"},
			expected: "MyProject",
		},
		{
			name:     "upperCase",
			tmpl:     `{{ upperCase .Name }}`,
			ctx:      map[string]string{"Name": "hello"},
			expected: "HELLO",
		},
		{
			name:     "join",
			tmpl:     `{{ join ", " .Items }}`,
			ctx:      map[string]interface{}{"Items": []string{"a", "b", "c"}},
			expected: "a, b, c",
		},
		{
			name:     "ternary in template",
			tmpl:     `{{ ternary "yes" "no" .Flag }}`,
			ctx:      map[string]interface{}{"Flag": true},
			expected: "yes",
		},
		{
			name:     "default in template",
			tmpl:     `{{ default "fallback" .Val }}`,
			ctx:      map[string]interface{}{"Val": ""},
			expected: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.RenderString(tt.tmpl, tt.ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderString_InvalidTemplate(t *testing.T) {
	r := NewRenderer()

	_, err := r.RenderString("{{ .Name", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse template")
}

func TestRenderString_ExecuteError(t *testing.T) {
	r := NewRenderer()

	// Calling a function with wrong arg type triggers an execution error
	_, err := r.RenderString("{{ join .Name .Name }}", map[string]interface{}{"Name": "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execute template")
}

func TestRendererWithDelimiters(t *testing.T) {
	r := NewRendererWithDelimiters("[[", "]]")

	ctx := map[string]string{"Name": "ironplate"}

	// [[ ]] delimiters should be processed
	result, err := r.RenderString("[[ .Name ]]", ctx)
	require.NoError(t, err)
	assert.Equal(t, "ironplate", result)

	// {{ }} should pass through unchanged since they are not delimiters
	result, err = r.RenderString("{{ .Values.image }} [[ .Name ]]", ctx)
	require.NoError(t, err)
	assert.Equal(t, "{{ .Values.image }} ironplate", result)
}

func TestRenderFile(t *testing.T) {
	r := NewRenderer()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.txt")

	tmplContent := []byte("Project: {{ .Name }}")
	ctx := map[string]string{"Name": "ironplate"}

	err := r.RenderFile(tmplContent, outputPath, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "Project: ironplate", string(data))
}

func TestRenderFile_EmptyOutput(t *testing.T) {
	r := NewRenderer()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "should-not-exist.txt")

	// Template that produces only whitespace
	tmplContent := []byte(`{{ if false }}content{{ end }}`)
	ctx := map[string]interface{}{}

	err := r.RenderFile(tmplContent, outputPath, ctx)
	require.NoError(t, err)

	// File should not be created since output is empty
	_, err = os.Stat(outputPath)
	assert.True(t, os.IsNotExist(err), "file should not exist for empty template output")
}

func TestRenderFS(t *testing.T) {
	r := NewRenderer()

	ctx := map[string]string{"Name": "ironplate"}

	memFS := fstest.MapFS{
		"templates/readme.md.tmpl": &fstest.MapFile{
			Data: []byte("# {{ .Name }}"),
		},
		"templates/static.txt": &fstest.MapFile{
			Data: []byte("I am static content with {{ braces }}"),
		},
		"templates/.gitkeep": &fstest.MapFile{
			Data: []byte(""),
		},
		"templates/helm.yaml.tpl.tmpl": &fstest.MapFile{
			Data: []byte("image: {{ .Values.image }}\nproject: [[ .Name ]]"),
		},
	}

	tmpDir := t.TempDir()

	err := r.RenderFS(memFS, "templates", tmpDir, ctx)
	require.NoError(t, err)

	// .tmpl file should be rendered with .tmpl extension stripped
	data, err := os.ReadFile(filepath.Join(tmpDir, "readme.md"))
	require.NoError(t, err)
	assert.Equal(t, "# ironplate", string(data))

	// Non-.tmpl file should be copied verbatim
	data, err = os.ReadFile(filepath.Join(tmpDir, "static.txt"))
	require.NoError(t, err)
	assert.Equal(t, "I am static content with {{ braces }}", string(data))

	// .gitkeep should be copied to preserve empty directories in git
	_, err = os.Stat(filepath.Join(tmpDir, ".gitkeep"))
	assert.False(t, os.IsNotExist(err), ".gitkeep should be copied")

	// .tpl.tmpl file should use [[ ]] delimiters, preserving {{ }}
	data, err = os.ReadFile(filepath.Join(tmpDir, "helm.yaml.tpl"))
	require.NoError(t, err)
	assert.Equal(t, "image: {{ .Values.image }}\nproject: ironplate", string(data))
}

func TestRenderFS_NestedDirs(t *testing.T) {
	r := NewRenderer()

	ctx := map[string]string{"Name": "ironplate"}

	memFS := fstest.MapFS{
		"root/subdir/file.txt.tmpl": &fstest.MapFile{
			Data: []byte("Hello {{ .Name }}"),
		},
		"root/subdir/deep/nested.txt": &fstest.MapFile{
			Data: []byte("static nested"),
		},
	}

	tmpDir := t.TempDir()

	err := r.RenderFS(memFS, "root", tmpDir, ctx)
	require.NoError(t, err)

	// Verify nested rendered file
	data, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, "Hello ironplate", string(data))

	// Verify deeply nested static file
	data, err = os.ReadFile(filepath.Join(tmpDir, "subdir", "deep", "nested.txt"))
	require.NoError(t, err)
	assert.Equal(t, "static nested", string(data))

	// Verify directory structure exists
	assert.DirExists(t, filepath.Join(tmpDir, "subdir"))
	assert.DirExists(t, filepath.Join(tmpDir, "subdir", "deep"))
}

func TestRenderFS_ExcludeDirs(t *testing.T) {
	r := NewRenderer()

	ctx := map[string]string{"Name": "ironplate"}

	memFS := fstest.MapFS{
		"root/src/index.ts.tmpl": &fstest.MapFile{
			Data: []byte("export const name = '{{ .Name }}'"),
		},
		"root/helm/Chart.yaml.tmpl": &fstest.MapFile{
			Data: []byte("name: {{ .Name }}"),
		},
		"root/helm/templates/deploy.yaml": &fstest.MapFile{
			Data: []byte("kind: Deployment"),
		},
	}

	tmpDir := t.TempDir()

	err := r.RenderFS(memFS, "root", tmpDir, ctx, "helm")
	require.NoError(t, err)

	// Source file should exist
	data, err := os.ReadFile(filepath.Join(tmpDir, "src", "index.ts"))
	require.NoError(t, err)
	assert.Equal(t, "export const name = 'ironplate'", string(data))

	// Excluded helm directory and its contents should not exist
	assert.NoDirExists(t, filepath.Join(tmpDir, "helm"))
	_, err = os.Stat(filepath.Join(tmpDir, "helm", "Chart.yaml"))
	assert.True(t, os.IsNotExist(err), "excluded file should not exist")
}
