package engine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MyProject", "my-project"},
		{"myProject", "my-project"},
		{"my-project", "my-project"},
		{"my_project", "my-project"},
		{"MyHTTPServer", "my-httpserver"},
		{"simple", "simple"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toKebabCase(tt.input))
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"MyProject", "my_project"},
		{"myProject", "my_project"},
		{"my-project", "my_project"},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toSnakeCase(tt.input))
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-project", "myProject"},
		{"my_project", "myProject"},
		{"MyProject", "myProject"},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toCamelCase(tt.input))
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-project", "MyProject"},
		{"my_project", "MyProject"},
		{"myProject", "MyProject"},
		{"simple", "Simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, toPascalCase(tt.input))
		})
	}
}

func TestHasItem(t *testing.T) {
	items := []string{"kafka", "redis", "hasura"}

	assert.True(t, hasItem(items, "kafka"))
	assert.True(t, hasItem(items, "redis"))
	assert.False(t, hasItem(items, "mongodb"))
	assert.False(t, hasItem(nil, "kafka"))
}

func TestDict(t *testing.T) {
	result := dict("name", "test", "version", 1)

	assert.Equal(t, "test", result["name"])
	assert.Equal(t, 1, result["version"])
}

func TestIndent(t *testing.T) {
	input := "line1\nline2\nline3"
	expected := "    line1\n    line2\n    line3"
	assert.Equal(t, expected, indent(4, input))
}

func TestDefaultVal(t *testing.T) {
	assert.Equal(t, "fallback", defaultVal("fallback", ""))
	assert.Equal(t, "value", defaultVal("fallback", "value"))
	assert.Equal(t, 42, defaultVal(42, 0))
	assert.Equal(t, 7, defaultVal(42, 7))
}

func TestTernary(t *testing.T) {
	assert.Equal(t, "yes", ternary("yes", "no", true))
	assert.Equal(t, "no", ternary("yes", "no", false))
}

func TestToCamelCase_Empty(t *testing.T) {
	assert.Equal(t, "", toCamelCase(""))
}

func TestToYAML(t *testing.T) {
	result, err := toYAML(map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.Contains(t, result, "key: value")

	// Trailing newline should be trimmed
	assert.False(t, strings.HasSuffix(result, "\n"), "trailing newline should be trimmed")
}

func TestToJSON(t *testing.T) {
	result, err := toJSON(map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"key":"value"}`, result)
}

func TestToPrettyJSON(t *testing.T) {
	result, err := toPrettyJSON(map[string]int{"count": 42})
	assert.NoError(t, err)
	assert.Contains(t, result, "\"count\": 42")
	assert.Contains(t, result, "\n")
}

func TestNindent(t *testing.T) {
	result := nindent(2, "hello\nworld")
	assert.Equal(t, "\n  hello\n  world", result)
}

func TestList(t *testing.T) {
	result := list("a", 1, true)
	assert.Len(t, result, 3)
	assert.Equal(t, "a", result[0])
	assert.Equal(t, 1, result[1])
	assert.Equal(t, true, result[2])
}

func TestDict_OddPairs(t *testing.T) {
	// Odd number of args — last key has no value, should be ignored
	result := dict("a", 1, "orphan")
	assert.Equal(t, 1, result["a"])
	_, exists := result["orphan"]
	assert.False(t, exists)
}

func TestIronFuncMap(t *testing.T) {
	fm := IronFuncMap()
	// Verify key functions are registered
	for _, name := range []string{
		"kebabCase", "camelCase", "pascalCase", "snakeCase",
		"toYaml", "toJson", "toPrettyJson",
		"indent", "nindent", "join", "split",
		"hasItem", "dict", "list",
		"b64enc", "b64dec", "printf",
		"default", "ternary", "quote",
	} {
		assert.NotNil(t, fm[name], "expected function %q in funcmap", name)
	}
}
