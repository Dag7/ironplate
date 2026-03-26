# Ironplate

Scaffold production-grade Kubernetes development environments with a single command.

Ironplate (`iron`) encodes battle-tested patterns — devcontainers, k3d, Tilt, Helm, Pulumi, ArgoCD, GitHub Actions — into a CLI that generates complete project scaffolding so you don't have to copy-paste between projects.

## Install

```bash
# From source
go install github.com/dag7/ironplate/cmd/iron@latest

# Homebrew (once published)
brew install dag7/tap/iron
```

## Quick Start

```bash
# Create a new project interactively
iron init

# Or use flags for non-interactive mode
iron init --name my-platform --preset standard --provider gcp

# Generate a service
iron generate service auth-api --type node-api --group auth

# Add infrastructure components
iron add kafka
iron add redis

# Check project status
iron status
```

## Commands

| Command | Description |
|---------|-------------|
| `iron init` | Create a new project with interactive prompts or flags |
| `iron generate service` | Generate a new microservice (Node.js, Go, Next.js) |
| `iron generate package` | Generate a shared package |
| `iron add <component>` | Add an infrastructure component (kafka, redis, hasura, etc.) |
| `iron remove <component>` | Remove an infrastructure component |
| `iron doctor` | Check prerequisites and tool versions |
| `iron list components` | List available infrastructure components |
| `iron list services` | List services in the current project |
| `iron status` | Show project configuration and status |
| `iron validate` | Validate ironplate.yaml configuration |
| `iron completion` | Generate shell completion scripts |
| `iron version` | Show version information |

## What Gets Generated

Ironplate scaffolds a complete monorepo with:

- **Dev Environment** — Devcontainer, k3d cluster, Tilt hot-reload
- **Services** — Multi-stage Dockerfiles, Helm charts, health checks
- **Infrastructure** — Kafka, Redis, Hasura, Dapr, observability stack
- **IaC** — Pulumi for GCP (AWS/Azure planned), VPC, GKE, Cloud SQL, etc.
- **CI/CD** — GitHub Actions for lint, test, build, deploy
- **GitOps** — ArgoCD with image updater
- **AI** — CLAUDE.md with project-specific rules, Claude Code skills

## Shell Completion

```bash
# Zsh (add to ~/.zshrc)
source <(iron completion zsh)

# Bash (add to ~/.bashrc)
source <(iron completion bash)

# Fish
iron completion fish | source

# PowerShell
iron completion powershell | Out-String | Invoke-Expression
```

For persistent completion, see `iron completion --help` for per-shell instructions.

## Infrastructure Components

| Component | Description |
|-----------|-------------|
| `kafka` | Event streaming (Strimzi operator) |
| `hasura` | GraphQL engine with migrations |
| `redis` | In-memory cache (Bitnami) |
| `dapr` | Distributed application runtime |
| `observability` | OpenTelemetry, Tempo, Prometheus, Grafana |
| `external-secrets` | Kubernetes External Secrets Operator |
| `argocd` | GitOps continuous delivery |

## Cloud Providers

| Provider | Status |
|----------|--------|
| GCP | Supported (GKE, Cloud SQL, Memorystore, Artifact Registry) |
| AWS | Planned |
| Azure | Planned |

## Project Structure

A generated project looks like:

```
my-platform/
  .devcontainer/          # Dev container + k3d cluster config
  apps/                   # Microservices
  packages/               # Shared libraries
  k8s/helm/               # Helm charts (services + infra)
  iac/pulumi/             # Infrastructure as Code
  .github/workflows/      # CI/CD pipelines
  tilt/                   # Tilt service registry + utils
  Tiltfile                # Dev orchestration
  CLAUDE.md               # AI coding rules
  .claude/skills/         # Claude Code skills
  ironplate.yaml          # Project configuration
```

## Development

```bash
# Build
make build

# Install locally
make install

# Run tests
make test

# Run linter
make lint

# Run integration tests
make test-integration
```

## License

Apache-2.0
