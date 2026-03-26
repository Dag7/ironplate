// Package engine provides the template rendering engine for ironplate.
package engine

import (
	"encoding/json"
	"strings"
	"text/template"
	"unicode"

	"gopkg.in/yaml.v3"
)

// IronFuncMap returns the custom template function map for ironplate templates.
func IronFuncMap() template.FuncMap {
	return template.FuncMap{
		// Case conversion
		"kebabCase":  toKebabCase,
		"camelCase":  toCamelCase,
		"pascalCase": toPascalCase,
		"snakeCase":  toSnakeCase,
		"upperCase":  strings.ToUpper,
		"lowerCase":  strings.ToLower,
		"title":      strings.Title, //nolint:staticcheck

		// String utilities
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    strings.ReplaceAll,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"join":       func(sep string, items []string) string { return strings.Join(items, sep) },
		"split":      strings.Split,
		"quote":      func(s string) string { return `"` + s + `"` },

		// Serialization
		"toYaml":     toYAML,
		"toJson":     toJSON,
		"toPrettyJson": toPrettyJSON,

		// Indentation
		"indent":  indent,
		"nindent": nindent,

		// Collection helpers
		"hasItem": hasItem,
		"dict":    dict,
		"list":    list,

		// Conditional helpers
		"default": defaultVal,
		"ternary": ternary,
	}
}

// toKebabCase converts a string to kebab-case.
func toKebabCase(s string) string {
	return strings.ToLower(addSeparator(s, '-'))
}

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
	return strings.ToLower(addSeparator(s, '_'))
}

// toCamelCase converts a string to camelCase.
func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// toPascalCase converts a string to PascalCase.
func toPascalCase(s string) string {
	words := splitWords(s)
	var result strings.Builder
	for _, w := range words {
		if len(w) > 0 {
			result.WriteString(strings.ToUpper(w[:1]) + strings.ToLower(w[1:]))
		}
	}
	return result.String()
}

// addSeparator inserts a separator between words in a string.
func addSeparator(s string, sep byte) string {
	words := splitWords(s)
	return strings.Join(words, string(sep))
}

// splitWords splits a string into lowercase words.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		switch {
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if current.Len() > 0 {
				words = append(words, strings.ToLower(current.String()))
				current.Reset()
			}
		case unicode.IsUpper(r) && i > 0:
			prev := rune(s[i-1])
			if !unicode.IsUpper(prev) && prev != '-' && prev != '_' && prev != ' ' {
				if current.Len() > 0 {
					words = append(words, strings.ToLower(current.String()))
					current.Reset()
				}
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		words = append(words, strings.ToLower(current.String()))
	}
	return words
}

func toYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func toJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func toPrettyJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func indent(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func nindent(spaces int, s string) string {
	return "\n" + indent(spaces, s)
}

func hasItem(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}
	return false
}

func dict(pairs ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i+1 < len(pairs); i += 2 {
		if key, ok := pairs[i].(string); ok {
			m[key] = pairs[i+1]
		}
	}
	return m
}

func list(items ...interface{}) []interface{} {
	return items
}

func defaultVal(def, val interface{}) interface{} {
	if val == nil || val == "" || val == 0 || val == false {
		return def
	}
	return val
}

func ternary(trueVal, falseVal interface{}, condition bool) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}
