# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Handoff is a Go library for bootstrapping servers that run scheduled and manually-triggered e2e tests. It includes a built-in web UI, REST API, cron scheduling, Prometheus metrics, and extensible hooks (Slack, Elasticsearch, PagerDuty, GitHub).

## Build & Development

```bash
# Build and run
templ generate                              # Required: generate HTML templates first
go build ./cmd/example-server-bootstrap/
./example-server-bootstrap

# Live reload development
air                                         # Watches files, regenerates templ, rebuilds

# Local K8s dev cluster (requires Docker, Tilt, Kind)
kind create cluster --config=kind-config.yaml
tilt up                                     # UI at localhost:1337 when green
```

## Testing

```bash
# Unit tests
go test ./...

# Run single test
go test -run TestName ./path/to/package

# K8s operator tests (from cmd/k8s-operator/)
make test                                   # Unit tests with envtest
make test-e2e                               # E2E tests with Kind
make lint                                   # golangci-lint
```

## K8s Operator (cmd/k8s-operator/)

```bash
make manifests    # Generate CRDs, RBAC, webhooks
make generate     # Generate DeepCopy methods
make build        # Build operator binary
make deploy       # Deploy to cluster
```

## Architecture

**Core Library (root):**
- `handoff.go` - Server struct, core logic
- `http.go` - HTTP server, router
- `hook_manager.go` - Plugin system
- `t.go` - Test interface definitions

**Key Directories:**
- `internal/html/` - Web UI templates (Templ framework, `*.templ` files)
- `internal/hook/` - Integration hooks (Slack, ES, PagerDuty, GitHub)
- `internal/storage/` - BadgerDB embedded database
- `internal/model/` - Data structures
- `client/` - Go HTTP client library
- `cmd/k8s-operator/` - Kubernetes operator for CRD-based config
- `chart/` - Helm chart

## Design Constraints (from CONTRIBUTING.md)

- **Single binary**: No external DB, cache, or frontend server
- **No horizontal scaling**: Single instance only
- **Keep symbols unexported** unless they need to be public API
- **Fast startup**: Don't add slow initialization
- **Minimal dependencies**: Only import what's absolutely needed

## Test Writing Guidelines

- Pass test context to long operations, check for cancellation
- Log only via `t.Log`/`t.Logf` (other logs won't appear in test output)
- Setup code must be idempotent (may run multiple times)
