# Compliance Operator Dashboard

Web dashboard for the [OpenShift Compliance Operator](https://github.com/ComplianceAsCode/compliance-operator) that provides full lifecycle management from a single UI: install the operator, run scans, view results, apply remediations, and uninstall.

Companion project to [compliance-scripts](https://github.com/sebrandon1/compliance-scripts) -- the same workflows reimplemented natively in Go with a React frontend, shipped as a single binary.

## Features

- **Operator Install / Uninstall** -- One-click install with real-time WebSocket progress. Auto-detects Red Hat certified vs community operator.
- **Scan Management** -- Create scans from any profile, run recommended suites (CIS, NIST 800-53, PCI-DSS), rescan, and delete.
- **Results & Remediation** -- Severity filtering, search, per-check rationale, and one-click remediation apply.
- **Real-Time Updates** -- WebSocket-driven live streaming of scan status, results, and remediation outcomes.
- **Single Binary** -- Go backend with embedded React SPA via `go:embed`.

## Quick Start

```bash
make build
./bin/compliance-operator-dashboard serve
```

Dashboard starts at [http://localhost:8080](http://localhost:8080) using your current kubeconfig.

## Commands

```bash
make build             # Build frontend + Go binary
make run               # Build and run
make test              # Run Go unit tests
make lint              # Run golangci-lint
make frontend-dev      # Start Vite dev server (port 5173, proxies API to :8080)
make clean             # Remove build artifacts
```

## Guides

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/guide/getting-started.md) | Step-by-step walkthrough with screenshots |
| [Configuration](docs/configuration.md) | CLI flags, environment variables, examples |
| [API Reference](docs/api.md) | REST endpoints and WebSocket |
| [Architecture](docs/architecture.md) | Project layout, key patterns, operator versioning |

## Requirements

- Go 1.22+
- Node.js 18+
- Access to an OpenShift cluster with kubeconfig configured
