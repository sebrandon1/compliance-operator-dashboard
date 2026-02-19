# Compliance Operator Dashboard

Web dashboard for the [OpenShift Compliance Operator](https://github.com/ComplianceAsCode/compliance-operator) that provides full lifecycle management from a single UI: install the operator, run scans, view results, apply remediations, and uninstall.

This is the companion project to [compliance-scripts](https://github.com/sebrandon1/compliance-scripts) — the same workflows those shell and Python scripts provide are reimplemented here natively in Go with a React frontend, shipped as a single binary.

## Features

- **Operator Install / Uninstall** — One-click install with real-time progress via WebSocket. Auto-detects Red Hat certified vs community operator. Full uninstall with cleanup of all CRs, subscriptions, and namespace.
- **Scan Management** — Create scans from any installed profile, run recommended scan suites (CIS, NIST 800-53 Moderate, PCI-DSS), rescan existing suites, and delete scans.
- **Results & Remediation** — View compliance check results with severity filtering and search. Inspect individual checks with rationale and instructions. Apply remediations with one click.
- **Real-Time Updates** — WebSocket-driven live updates from Kubernetes watch events. Scan status, check results, and remediation outcomes stream to the browser automatically.
- **Single Binary** — Go backend with embedded React SPA via `go:embed`. No separate frontend server needed.

## Quick Start

```bash
# Build everything (frontend + Go binary)
make build

# Run the dashboard
./bin/compliance-operator-dashboard serve
```

The dashboard starts at [http://localhost:8080](http://localhost:8080) and connects to the cluster via your current kubeconfig.

## Commands

```bash
make build             # Build frontend + Go binary
make run               # Build and run
make test              # Run Go unit tests
make lint              # Run golangci-lint
make frontend-dev      # Start Vite dev server (port 5173, proxies API to :8080)
make frontend-install  # npm install in frontend/
make frontend-build    # Build frontend to frontend/dist/
make clean             # Remove build artifacts
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--kubeconfig` | `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig file |
| `--namespace` | `COMPLIANCE_NAMESPACE` | `openshift-compliance` | Namespace for compliance resources |
| `--port` | — | `8080` | HTTP server port |
| `--co-ref` | `COMPLIANCE_OPERATOR_REF` | latest from GitHub | Compliance Operator version (community install only) |

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cluster/status` | Cluster connectivity, version, architecture |
| `POST` | `/api/operator/install` | Start operator installation |
| `GET` | `/api/operator/status` | Current operator status |
| `DELETE` | `/api/operator` | Uninstall operator |
| `POST` | `/api/scans` | Create a scan |
| `GET` | `/api/scans` | List all scan suites |
| `POST` | `/api/scans/recommended` | Create recommended scan suites |
| `POST` | `/api/scans/{name}/rescan` | Trigger rescan of a suite |
| `DELETE` | `/api/scans/{name}` | Delete a scan suite |
| `GET` | `/api/profiles` | List available compliance profiles |
| `GET` | `/api/results` | Get compliance results (supports `severity`, `status`, `search` query params) |
| `GET` | `/api/results/summary` | Summary counts |
| `GET` | `/api/results/{name}` | Detail for a single check result |
| `GET` | `/api/remediations` | List all remediations |
| `GET` | `/api/remediations/{name}` | Detail for a single remediation |
| `POST` | `/api/remediate/{name}` | Apply a remediation |
| `GET` | `/ws/watch` | WebSocket for real-time updates |

## Architecture

```
main.go + cmd/           Cobra CLI with "serve" subcommand
internal/config/         Configuration (flags, env vars)
internal/k8s/            Kubernetes client (typed + dynamic)
internal/compliance/     Core logic:
  operator.go              Install, uninstall, status
  scan.go                  Create, rescan, delete scans
  results.go               Collect and filter results
  remediation.go           Apply remediations
  storage.go               Storage class detection
internal/api/            HTTP server, REST handlers, middleware
internal/ws/             WebSocket hub, K8s watch bridge
frontend/                React 18 + TypeScript + Vite + Tailwind + Zustand
```

## Operator Versioning

There are two distribution channels with **different version numbers**:

- **Red Hat certified** (`redhat-operators` catalog) — Versioned independently by Red Hat (e.g., v1.8.2). Built internally, not publicly tagged on GitHub. The old downstream repo at [openshift/compliance-operator](https://github.com/openshift/compliance-operator) is deprecated.
- **Upstream/community** ([ComplianceAsCode/compliance-operator](https://github.com/ComplianceAsCode/compliance-operator)) — Latest release is v1.7.0. Used when `redhat-operators` is not available on the cluster.

The dashboard auto-detects which source to use. If the cluster has `redhat-operators` in `openshift-marketplace`, it installs the Red Hat certified version. Otherwise it uses the community catalog image from `ghcr.io`. The `--co-ref` flag only applies to the community install path.

## Related Projects

| Repository | Description |
|------------|-------------|
| [compliance-scripts](https://github.com/sebrandon1/compliance-scripts) | Shell/Python scripts for the same compliance workflows. The dashboard reimplements these as a web UI. Sister repo — features should be paired. |
| [ComplianceAsCode/compliance-operator](https://github.com/ComplianceAsCode/compliance-operator) | Upstream Compliance Operator |

## Requirements

- Go 1.22+
- Node.js 18+
- Access to an OpenShift cluster with `oc` / kubeconfig configured
