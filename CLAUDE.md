# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Web dashboard for OpenShift Compliance Operator that reimplements compliance-scripts shell/Python workflows natively in Go with a React + TypeScript frontend. Single Go binary with embedded React SPA provides operator install, scan execution, result visualization, and one-click remediation with real-time WebSocket updates.

## Common Commands

```bash
# Build everything (frontend + backend)
make build

# Development
make frontend-dev          # Start Vite dev server (port 5173, proxies to :8080)
make run                   # Build and run the Go binary

# Testing
make test                  # Run Go unit tests
make lint                  # Run golangci-lint

# Individual steps
make frontend-install      # npm install in frontend/
make frontend-build        # Build frontend to frontend/dist/
make clean                 # Remove build artifacts
```

## Architecture

- `main.go` + `cmd/` - Cobra CLI with `serve` subcommand
- `internal/config/` - Configuration struct, flag/env loading
- `internal/k8s/` - Kubernetes client (clientcmd, typed + dynamic)
- `internal/compliance/` - Core logic reimplementing compliance-scripts:
  - `operator.go` - Install/status (from install-compliance-operator.sh)
  - `scan.go` - Create/schedule scans (from create-scan.sh, apply-periodic-scan.sh)
  - `results.go` - Collect results (from export-compliance-data.sh)
  - `remediation.go` - Apply remediations (from apply-remediations-by-severity.sh)
  - `storage.go` - Storage class detection
- `internal/api/` - HTTP server with go:embed, REST handlers, middleware
- `internal/ws/` - WebSocket hub, client, K8s watch bridge
- `frontend/` - React 18 + TypeScript + Vite + Tailwind + Zustand

## Key Patterns

- All K8s operations use `context.Context` with timeouts
- Dynamic client for Compliance Operator CRDs (unstructured)
- Typed client for core K8s resources (pods, namespaces, RBAC)
- WebSocket hub broadcasts K8s watch events to all connected browsers
- Frontend uses Zustand for state, axios for API calls, custom WebSocket hook
- Single binary: `go:embed all:frontend/dist` serves the React SPA

## Operator Versioning

There are two distribution channels for the Compliance Operator with **different version numbers**:

- **Red Hat certified** (`redhat-operators` catalog): Versioned independently by Red Hat (e.g., v1.8.2). The source for these builds is internal to Red Hat's build system and not publicly tagged on GitHub. The old downstream repo at [openshift/compliance-operator](https://github.com/openshift/compliance-operator) is deprecated.
- **Upstream/community** ([ComplianceAsCode/compliance-operator](https://github.com/ComplianceAsCode/compliance-operator)): Latest release is v1.7.0. Used when `redhat-operators` is not available on the cluster.

The dashboard auto-detects which source to use: if the cluster has `redhat-operators` in `openshift-marketplace`, it installs the Red Hat certified version (which may have a higher version number than the upstream). Otherwise it falls back to the community catalog image from `ghcr.io`. The `--co-ref` flag only applies to the community install path.

## Flags

- `--kubeconfig` / `$KUBECONFIG` / `~/.kube/config`
- `--namespace` / `$COMPLIANCE_NAMESPACE` / `openshift-compliance`
- `--port` / `8080`
- `--co-ref` / `$COMPLIANCE_OPERATOR_REF` / latest from GitHub (community install only)
