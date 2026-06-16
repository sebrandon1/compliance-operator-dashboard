# Configuration

The dashboard accepts configuration via CLI flags and environment variables.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--kubeconfig` | `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig file |
| `--namespace` | `COMPLIANCE_NAMESPACE` | `openshift-compliance` | Namespace for compliance resources |
| `--port` | — | `8080` | HTTP server port |
| `--co-ref` | `COMPLIANCE_OPERATOR_REF` | latest from GitHub | Compliance Operator version (community install only) |

## Examples

```bash
# Use a specific kubeconfig
./bin/compliance-operator-dashboard serve --kubeconfig=/path/to/kubeconfig

# Run on a different port in a custom namespace
./bin/compliance-operator-dashboard serve --port=9090 --namespace=my-compliance

# Pin a specific community operator version
COMPLIANCE_OPERATOR_REF=v1.7.0 ./bin/compliance-operator-dashboard serve
```
