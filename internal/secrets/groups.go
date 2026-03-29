// Package secrets defines credential groups and provides secrets management
// for ironplate projects with Pulumi IaC.
package secrets

import "github.com/dag7/ironplate/internal/config"

// FieldType indicates how a secret field value is obtained.
type FieldType int

const (
	// FieldManual requires the user to provide a value.
	FieldManual FieldType = iota
	// FieldGenerate can be auto-generated (random secret).
	FieldGenerate
	// FieldDerived is computed from infrastructure (DB URLs, etc.) — skip in interactive mode.
	FieldDerived
)

// GeneratorKind identifies which random generator to use for FieldGenerate fields.
type GeneratorKind string

const (
	GenJWTSecret     GeneratorKind = "jwt-secret"     // 64-char alphanumeric
	GenEncryptionKey GeneratorKind = "encryption-key"  // 32-char alphanumeric
	GenPassword      GeneratorKind = "password"        // 32-char with special chars
	GenAPIKey        GeneratorKind = "api-key"         // 48-char alphanumeric
	GenCookieSecret  GeneratorKind = "cookie-secret"   // 32-char alphanumeric
)

// Field describes a single secret within a credential group.
type Field struct {
	Key         string        // JSON key (e.g., "jwt-secret")
	Description string        // Human-readable description
	Type        FieldType     // How value is obtained
	Generator   GeneratorKind // Which generator (only for FieldGenerate)
	Placeholder string        // Default placeholder value in template
}

// Group describes a credential group (maps to a single Pulumi config secret).
type Group struct {
	Name        string   // Group name (e.g., "jwt-credentials")
	Description string   // Human-readable description
	Fields      []Field  // Fields within this group
	Condition   string   // Component condition (empty = always included)
}

// AllGroups returns all credential groups, in the order they should be presented.
func AllGroups() []Group {
	return []Group{
		{
			Name:        "jwt-credentials",
			Description: "JWT authentication tokens",
			Fields: []Field{
				{Key: "jwt-secret", Description: "JWT signing secret (64-char)", Type: FieldGenerate, Generator: GenJWTSecret},
				{Key: "jwt-issuer", Description: "JWT issuer identifier", Type: FieldManual, Placeholder: "https://auth.example.com"},
				{Key: "jwt-audience", Description: "JWT audience identifier", Type: FieldManual, Placeholder: "https://api.example.com"},
			},
		},
		{
			Name:        "encryption-credentials",
			Description: "Data encryption keys",
			Fields: []Field{
				{Key: "encryption-key", Description: "AES-256 encryption key (32-char)", Type: FieldGenerate, Generator: GenEncryptionKey},
			},
		},
		{
			Name:        "hasura-credentials",
			Description: "Hasura GraphQL Engine secrets",
			Condition:   "hasura",
			Fields: []Field{
				{Key: "hasura-admin-secret", Description: "Hasura admin secret", Type: FieldGenerate, Generator: GenPassword},
				{Key: "hasura-graphql-jwt-secret", Description: "Hasura JWT config JSON", Type: FieldManual, Placeholder: `{"type":"HS256","key":"...","claims_namespace":"https://hasura.io/jwt/claims"}`},
				{Key: "hasura-webhook-secret", Description: "Hasura event trigger webhook secret", Type: FieldGenerate, Generator: GenPassword},
			},
		},
		{
			Name:        "oauth-credentials",
			Description: "OAuth provider credentials",
			Fields: []Field{
				{Key: "google-client-id", Description: "Google OAuth client ID", Type: FieldManual},
				{Key: "google-client-secret", Description: "Google OAuth client secret", Type: FieldManual},
				{Key: "google-authorization-url", Description: "Google authorization URL", Type: FieldManual, Placeholder: "https://accounts.google.com/o/oauth2/v2/auth"},
				{Key: "google-token-url", Description: "Google token URL", Type: FieldManual, Placeholder: "https://oauth2.googleapis.com/token"},
				{Key: "google-user-url", Description: "Google userinfo URL", Type: FieldManual, Placeholder: "https://www.googleapis.com/oauth2/v2/userinfo"},
				{Key: "github-client-id", Description: "GitHub OAuth client ID", Type: FieldManual},
				{Key: "github-client-secret", Description: "GitHub OAuth client secret", Type: FieldManual},
			},
		},
		{
			Name:        "llm-credentials",
			Description: "LLM API keys",
			Fields: []Field{
				{Key: "openai-api-key", Description: "OpenAI API key", Type: FieldManual},
				{Key: "anthropic-api-key", Description: "Anthropic API key", Type: FieldManual},
				{Key: "google-api-key", Description: "Google AI API key", Type: FieldManual},
				{Key: "ollama-base-url", Description: "Ollama server URL", Type: FieldManual, Placeholder: "http://localhost:11434"},
			},
		},
		{
			Name:        "langfuse-credentials",
			Description: "Langfuse LLM observability",
			Condition:   "langfuse",
			Fields: []Field{
				{Key: "langfuse-nextauth-secret", Description: "NextAuth secret for Langfuse", Type: FieldGenerate, Generator: GenJWTSecret},
				{Key: "langfuse-salt", Description: "Langfuse salt", Type: FieldGenerate, Generator: GenEncryptionKey},
				{Key: "langfuse-encryption-key", Description: "Langfuse encryption key", Type: FieldGenerate, Generator: GenEncryptionKey},
				{Key: "langfuse-public-key", Description: "Langfuse public key", Type: FieldManual},
				{Key: "langfuse-secret-key", Description: "Langfuse secret key", Type: FieldManual},
				{Key: "langfuse-base-url", Description: "Langfuse base URL", Type: FieldManual},
				{Key: "langfuse-init-user-password", Description: "Langfuse initial admin password", Type: FieldGenerate, Generator: GenPassword},
			},
		},
		{
			Name:        "langfuse-clickhouse-credentials",
			Description: "Langfuse ClickHouse credentials",
			Condition:   "langfuse",
			Fields: []Field{
				{Key: "clickhouse-user", Description: "ClickHouse username", Type: FieldManual, Placeholder: "default"},
				{Key: "clickhouse-password", Description: "ClickHouse password", Type: FieldGenerate, Generator: GenPassword},
			},
		},
		{
			Name:        "statsig-credentials",
			Description: "Statsig feature flags",
			Fields: []Field{
				{Key: "statsig-server-key", Description: "Statsig server-side key", Type: FieldManual},
				{Key: "statsig-client-key", Description: "Statsig client-side key", Type: FieldManual},
			},
		},
		{
			Name:        "email-credentials",
			Description: "Email service (Resend)",
			Fields: []Field{
				{Key: "resend-api-key", Description: "Resend API key", Type: FieldManual},
			},
		},
		{
			Name:        "iap-credentials",
			Description: "Identity-Aware Proxy (auto-managed by setup script)",
			Fields: []Field{
				{Key: "client-id", Description: "IAP OAuth client ID", Type: FieldManual},
				{Key: "client-secret", Description: "IAP OAuth client secret", Type: FieldManual},
				{Key: "cookie-secret", Description: "OAuth2 Proxy cookie secret", Type: FieldGenerate, Generator: GenCookieSecret},
			},
		},
		{
			Name:        "github-app-credentials",
			Description: "GitHub App for ArgoCD",
			Condition:   "argocd",
			Fields: []Field{
				{Key: "app-id", Description: "GitHub App ID", Type: FieldManual},
				{Key: "installation-id", Description: "GitHub App installation ID", Type: FieldManual},
				{Key: "private-key", Description: "GitHub App private key (PEM)", Type: FieldManual},
				{Key: "client-id", Description: "GitHub App client ID", Type: FieldManual},
				{Key: "client-secret", Description: "GitHub App client secret", Type: FieldManual},
			},
		},
		{
			Name:        "slack-credentials",
			Description: "Slack bot for notifications",
			Fields: []Field{
				{Key: "slack-bot-token", Description: "Slack bot OAuth token", Type: FieldManual},
			},
		},
		{
			Name:        "grafana-credentials",
			Description: "Grafana monitoring dashboard",
			Condition:   "observability",
			Fields: []Field{
				{Key: "admin-user", Description: "Grafana admin username", Type: FieldManual, Placeholder: "admin"},
				{Key: "admin-password", Description: "Grafana admin password", Type: FieldGenerate, Generator: GenPassword},
			},
		},
		{
			Name:        "alertmanager-credentials",
			Description: "Alert routing (Slack/PagerDuty)",
			Condition:   "observability",
			Fields: []Field{
				{Key: "slack-webhook-url", Description: "Slack incoming webhook URL", Type: FieldManual},
				{Key: "pagerduty-service-key", Description: "PagerDuty service integration key", Type: FieldManual},
			},
		},
		{
			Name:        "stripe-credentials",
			Description: "Stripe payment processing",
			Fields: []Field{
				{Key: "secret-key", Description: "Stripe secret key", Type: FieldManual},
				{Key: "publishable-key", Description: "Stripe publishable key", Type: FieldManual},
				{Key: "webhook-secret", Description: "Stripe webhook signing secret", Type: FieldManual},
			},
		},
		{
			Name:        "visitor-credentials",
			Description: "Visitor session management",
			Fields: []Field{
				{Key: "cookie-secret", Description: "Visitor cookie secret", Type: FieldGenerate, Generator: GenCookieSecret},
			},
		},
		{
			Name:        "search-provider-credentials",
			Description: "Search API provider",
			Fields: []Field{
				{Key: "brave-search-api-key", Description: "Brave Search API key", Type: FieldManual},
			},
		},
	}
}

// GroupsForConfig returns only the credential groups applicable to the given project config.
func GroupsForConfig(cfg *config.ProjectConfig) []Group {
	infra := cfg.Spec.Infrastructure
	var result []Group
	for _, g := range AllGroups() {
		if g.Condition == "" || infra.HasComponent(g.Condition) {
			result = append(result, g)
		}
	}
	return result
}
