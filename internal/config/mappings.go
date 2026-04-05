package config

// FeatureComponentMap maps service features to the infrastructure components they require.
var FeatureComponentMap = map[string]string{
	"hasura":   "hasura",
	"cache":    "redis",
	"dapr":     "dapr",
	"eventbus": "kafka",
}

// TypeLanguageMap maps service types to the language they require.
var TypeLanguageMap = map[string]string{
	"node-api":  "node",
	"nextjs":    "node",
	"go-api":    "go",
	"go-worker": "go",
}

// SupportedServiceTypes lists all valid service types.
var SupportedServiceTypes = []string{"node-api", "go-api", "nextjs"}

// ServiceTemplateDirs maps service types to their template directories.
var ServiceTemplateDirs = map[string]string{
	"node-api": "service/node",
	"go-api":   "service/go",
	"nextjs":   "service/nextjs",
}
