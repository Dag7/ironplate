# Ironplate

> Scaffold production-grade Kubernetes development environments with a single command.

Ironplate (`iron`) encodes battle-tested patterns — devcontainers, k3d, Tilt, Helm, Pulumi, ArgoCD, GitHub Actions — into a CLI that generates complete project scaffolding so you don't have to copy-paste between projects.

**One command. Full stack. Ready to `tilt up`.**

```bash
iron init
```

---

## Install

```bash
# From source
go install github.com/dag7/ironplate/cmd/iron@latest

# Homebrew (once published)
brew install dag7/tap/iron
```

## Quick Start

```bash
# Interactive — walks you through every choice
iron init

# Non-interactive — CI-friendly
iron init --name my-platform --org myorg --preset standard --provider gcp --non-interactive

# Add services
iron generate service auth-api --type node-api --group auth
iron generate service gateway --type go-api --group core

# Add infrastructure
iron add kafka
iron add redis

# Check everything
iron doctor
iron status
```

## Commands

| Command | Description |
|---------|-------------|
| `iron init` | Create a new project (interactive or flags) |
| `iron generate service` | Add a microservice (Node.js, Go, Next.js) |
| `iron generate package` | Add a shared library package |
| `iron add <component>` | Install an infrastructure component |
| `iron remove <component>` | Remove an infrastructure component |
| `iron doctor` | Verify prerequisites and tool versions |
| `iron list components` | Browse available components |
| `iron list services` | List project services |
| `iron status` | Show project configuration |
| `iron validate` | Validate `ironplate.yaml` |
| `iron completion` | Generate shell completion scripts |
| `iron version` | Print version info |

## What Gets Generated

```
my-platform/
  .devcontainer/              Dev container + k3d cluster
  apps/                       Microservices
  packages/                   Shared libraries (node + go)
  k8s/helm/                   Helm charts (services + infra)
  iac/pulumi/                 Infrastructure as Code
  .github/workflows/          CI/CD pipelines
  utils/tilt/                 Tilt build utilities
  Tiltfile                    Dev orchestration entry point
  my-platform.code-workspace  VS Code multi-root workspace
  CLAUDE.md                   AI coding rules (assembled per-project)
  .claude/skills/             Claude Code skills
  ironplate.yaml              Project manifest
```

See [docs/generated-project.md](docs/generated-project.md) for a detailed breakdown.

## Infrastructure Components

Components are pluggable — add what you need, skip what you don't.

| Component | What you get |
|-----------|-------------|
| `kafka` | Strimzi Kafka operator, KafkaUI, topic CRDs |
| `hasura` | GraphQL engine, migrations, metadata, codegen |
| `redis` | Bitnami Redis with RedisInsight |
| `dapr` | Distributed runtime (pub/sub, state, bindings) |
| `observability` | OpenTelemetry + Tempo + Prometheus + Grafana |
| `external-secrets` | K8s External Secrets Operator (cloud secret sync) |
| `argocd` | GitOps CD with image updater |
| `langfuse` | LLM observability platform |

Dependencies are resolved automatically — `argocd` pulls in `external-secrets`, etc.

## Dev Tools

During `iron init`, you can select optional dev container tools:

| Tool | Purpose |
|------|---------|
| `operator-sdk` | Build Kubernetes operators |
| `git-secret` | Encrypt secrets in git |
| `mc` | MinIO/S3-compatible object storage CLI |
| `kompose` | Convert Docker Compose to K8s manifests |

All selected tools get shell completions auto-configured.

**Always included:** kubectl, helm, k3d, tilt, krew, GitHub CLI, color-coded K8s context prompt.

## Cloud Providers

| Provider | Status | What's generated |
|----------|--------|-----------------|
| **GCP** | Supported | GKE, Cloud SQL, Memorystore, Artifact Registry, Cloud Armor, VPC |
| **AWS** | Planned | EKS, RDS, ElastiCache, ECR |
| **Azure** | Planned | AKS, Azure SQL, Azure Cache, ACR |

IaC uses Pulumi with a [12-phase orchestrator](docs/iac.md) pattern.

## AI Integration

Ironplate generates project-aware AI configuration:

- **CLAUDE.md** — Assembled from conditional sections (coding standards, Helm rules, GraphQL patterns, caching conventions, etc.) based on your selected components
- **Skills** — Claude Code slash commands for common tasks (`/new-service`, `/new-migration`, `/setup-cache`, etc.)

Only relevant sections are included — a project without Hasura won't get GraphQL rules.

## Shell Completion

```bash
# Zsh (add to ~/.zshrc)
source <(iron completion zsh)

# Bash
source <(iron completion bash)

# Fish
iron completion fish | source

# PowerShell
iron completion powershell | Out-String | Invoke-Expression
```

## Configuration

Projects are configured via `ironplate.yaml` — the manifest that tracks languages, components, services, and cloud settings. See [docs/configuration.md](docs/configuration.md) for the full schema.

```yaml
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: my-platform
  organization: myorg
spec:
  languages: [node, go]
  cloud:
    provider: gcp
  infrastructure:
    components: [kafka, hasura, redis, observability]
  devEnvironment:
    tools: [operator-sdk, git-secret]
```

## Development

```bash
make build            # Build the iron binary
make install          # Install to $GOPATH/bin
make test             # Run unit tests
make test-integration # Run integration tests
make lint             # Run golangci-lint
make check            # Run all checks (fmt, vet, lint, test)
```

See [CLAUDE.md](CLAUDE.md) for coding standards, architecture patterns, and contribution guidelines.

## License

Apache-2.0
