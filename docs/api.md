# API Reference

All REST endpoints are served under `/api/`. A WebSocket endpoint provides real-time updates.

## Cluster

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/cluster/status` | Cluster connectivity, version, architecture |

## Operator

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/operator/install` | Start operator installation |
| `GET` | `/api/operator/status` | Current operator status |
| `DELETE` | `/api/operator` | Uninstall operator |

## Scans

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/scans` | Create a scan |
| `GET` | `/api/scans` | List all scan suites |
| `POST` | `/api/scans/recommended` | Create recommended scan suites |
| `POST` | `/api/scans/{name}/rescan` | Trigger rescan of a suite |
| `DELETE` | `/api/scans/{name}` | Delete a scan suite |

## Profiles

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/profiles` | List available compliance profiles |

## Results

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/results` | Get compliance results (supports `severity`, `status`, `search` query params) |
| `GET` | `/api/results/summary` | Summary counts |
| `GET` | `/api/results/{name}` | Detail for a single check result |

## Remediations

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/remediations` | List all remediations |
| `GET` | `/api/remediations/{name}` | Detail for a single remediation |
| `POST` | `/api/remediate/{name}` | Apply a remediation |

## WebSocket

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/ws/watch` | WebSocket for real-time updates |
