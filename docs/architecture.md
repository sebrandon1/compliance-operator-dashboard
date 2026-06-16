# Architecture

The dashboard is a single Go binary with an embedded React SPA. No separate frontend server is needed in production.

## Project Layout

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

## Key Patterns

- All Kubernetes operations use `context.Context` with timeouts.
- Dynamic client for Compliance Operator CRDs (unstructured).
- Typed client for core Kubernetes resources (pods, namespaces, RBAC).
- WebSocket hub broadcasts Kubernetes watch events to all connected browsers.
- Frontend uses Zustand for state, axios for API calls, and a custom WebSocket hook.
- `go:embed all:frontend/dist` serves the React SPA from the compiled binary.

## Operator Versioning

There are two distribution channels with **different version numbers**:

- **Red Hat certified** (`redhat-operators` catalog) -- Versioned independently by Red Hat (e.g., v1.8.2). Built internally, not publicly tagged on GitHub. The old downstream repo at [openshift/compliance-operator](https://github.com/openshift/compliance-operator) is deprecated.
- **Upstream/community** ([ComplianceAsCode/compliance-operator](https://github.com/ComplianceAsCode/compliance-operator)) -- Latest release is v1.7.0. Used when `redhat-operators` is not available on the cluster.

The dashboard auto-detects which source to use. If the cluster has `redhat-operators` in `openshift-marketplace`, it installs the Red Hat certified version. Otherwise it uses the community catalog image from `ghcr.io`. The `--co-ref` flag only applies to the community install path.

## Related Projects

| Repository | Description |
|------------|-------------|
| [compliance-scripts](https://github.com/sebrandon1/compliance-scripts) | Shell/Python scripts for the same compliance workflows. The dashboard reimplements these as a web UI. |
| [ComplianceAsCode/compliance-operator](https://github.com/ComplianceAsCode/compliance-operator) | Upstream Compliance Operator |
