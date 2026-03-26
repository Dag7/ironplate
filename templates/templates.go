// Package templates embeds all ironplate template files into the binary.
package templates

import "embed"

// FS contains all embedded template files.
//
//go:embed all:base all:monorepo all:devcontainer all:k3d all:dockerfiles all:tilt all:k8s all:components all:iac all:cicd all:cli all:service all:package all:claude-md all:skills all:scripts
// Note: k8s/ contains Helm library chart, ingress, and local dev manifests
// components/ contains per-component Helm charts (kafka, hasura, etc.)
var FS embed.FS
