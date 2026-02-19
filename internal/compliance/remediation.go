package compliance

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

// ApplyRemediation applies a single ComplianceRemediation by extracting its
// spec.current.object and performing a server-side apply.
// Reimplements misc/apply-remediations-by-severity.sh single-item logic.
func ApplyRemediation(ctx context.Context, client *k8s.Client, namespace, name string) (*RemediationResult, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	result := &RemediationResult{Name: name}

	// Get the remediation
	rem, err := client.Dynamic.Resource(complianceRemediationGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		result.Error = fmt.Sprintf("getting remediation: %v", err)
		return result, fmt.Errorf("getting remediation %s: %w", name, err)
	}

	// Extract spec.current.object
	obj, found, err := unstructured.NestedMap(rem.Object, "spec", "current", "object")
	if err != nil || !found {
		result.Error = "remediation has no spec.current.object"
		return result, fmt.Errorf("remediation %s has no spec.current.object", name)
	}

	// Create an Unstructured from the extracted object
	remObj := &unstructured.Unstructured{Object: obj}
	kind := remObj.GetKind()
	apiVersion := remObj.GetAPIVersion()

	if kind == "" || apiVersion == "" {
		result.Error = "remediation object missing kind or apiVersion"
		return result, fmt.Errorf("remediation %s object missing kind or apiVersion", name)
	}

	// Determine the GVR for the remediation object
	gvr, objNamespace, err := resolveGVR(kind, apiVersion, namespace)
	if err != nil {
		result.Error = fmt.Sprintf("resolving GVR: %v", err)
		return result, err
	}

	// Prefer the object's own namespace over the resolved default
	if ns := remObj.GetNamespace(); ns != "" {
		objNamespace = ns
	}

	// Ensure metadata.name is set
	objName := remObj.GetName()
	if objName == "" {
		// Use the remediation name as the object name
		remObj.SetName(name)
		objName = name
	}

	// Apply the object
	if objNamespace != "" {
		_, err = client.Dynamic.Resource(gvr).Namespace(objNamespace).
			Create(ctx, remObj, metav1.CreateOptions{})
	} else {
		_, err = client.Dynamic.Resource(gvr).
			Create(ctx, remObj, metav1.CreateOptions{})
	}

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			// Update instead
			if objNamespace != "" {
				existing, getErr := client.Dynamic.Resource(gvr).Namespace(objNamespace).
					Get(ctx, objName, metav1.GetOptions{})
				if getErr == nil {
					remObj.SetResourceVersion(existing.GetResourceVersion())
				}
				_, err = client.Dynamic.Resource(gvr).Namespace(objNamespace).
					Update(ctx, remObj, metav1.UpdateOptions{})
			} else {
				existing, getErr := client.Dynamic.Resource(gvr).
					Get(ctx, objName, metav1.GetOptions{})
				if getErr == nil {
					remObj.SetResourceVersion(existing.GetResourceVersion())
				}
				_, err = client.Dynamic.Resource(gvr).
					Update(ctx, remObj, metav1.UpdateOptions{})
			}
		}
		if err != nil {
			result.Error = fmt.Sprintf("applying object: %v", err)
			return result, fmt.Errorf("applying remediation %s: %w", name, err)
		}
	}

	result.Applied = true
	result.Message = fmt.Sprintf("Applied %s %s", kind, objName)

	// If MachineConfig, add reboot hint
	if kind == "MachineConfig" {
		role := detectRoleFromObject(remObj)
		result.Message += fmt.Sprintf(" (MachineConfig - nodes with role %s will reboot)", role)
	}

	return result, nil
}

// ApplyBySeverity applies all remediations matching a given severity level.
// Reimplements misc/apply-remediations-by-severity.sh bulk logic.
func ApplyBySeverity(ctx context.Context, client *k8s.Client, namespace string, severity Severity, progress chan<- RemediationResult) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}
	defer close(progress)

	// List all remediations
	remediations, err := ListRemediations(ctx, client, namespace)
	if err != nil {
		return fmt.Errorf("listing remediations: %w", err)
	}

	for _, rem := range remediations {
		if rem.Severity != severity {
			continue
		}

		result, err := ApplyRemediation(ctx, client, namespace, rem.Name)
		if err != nil {
			progress <- RemediationResult{
				Name:  rem.Name,
				Error: err.Error(),
			}

			// Wait briefly between operations for MachineConfig to avoid overwhelming MCP
			if rem.Kind == "MachineConfig" {
				waitForMCPReconciliation(ctx, client, rem.Role)
			}
			continue
		}

		progress <- *result

		// Wait for MachineConfig changes to reconcile
		if rem.Kind == "MachineConfig" {
			waitForMCPReconciliation(ctx, client, rem.Role)
		}
	}

	return nil
}

func resolveGVR(kind, apiVersion, defaultNamespace string) (gvr schema.GroupVersionResource, namespace string, err error) {
	parts := strings.SplitN(apiVersion, "/", 2)
	var group, version string
	if len(parts) == 2 {
		group = parts[0]
		version = parts[1]
	} else {
		version = parts[0]
	}

	// Map common kinds to their resource names
	resourceName := strings.ToLower(kind) + "s"
	switch kind {
	case "MachineConfig":
		resourceName = "machineconfigs"
		group = "machineconfiguration.openshift.io"
		version = "v1"
		namespace = "" // Cluster-scoped
	case "APIServer":
		resourceName = "apiservers"
		group = "config.openshift.io"
		version = "v1"
		namespace = ""
	case "KubeletConfig":
		resourceName = "kubeletconfigs"
		group = "machineconfiguration.openshift.io"
		version = "v1"
		namespace = ""
	case "IngressController":
		resourceName = "ingresscontrollers"
		group = "operator.openshift.io"
		version = "v1"
	case "OAuth":
		resourceName = "oauths"
		group = "config.openshift.io"
		version = "v1"
		namespace = ""
	case "ConfigMap":
		resourceName = "configmaps"
		namespace = defaultNamespace
	case "Secret":
		resourceName = "secrets"
		namespace = defaultNamespace
	default:
		namespace = defaultNamespace
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourceName,
	}, namespace, nil
}

func detectRoleFromObject(obj *unstructured.Unstructured) string {
	labels := obj.GetLabels()
	if role, ok := labels["machineconfiguration.openshift.io/role"]; ok {
		return role
	}

	name := strings.ToLower(obj.GetName())
	if strings.Contains(name, "master") {
		return "master"
	}
	return "worker"
}

func waitForMCPReconciliation(ctx context.Context, client *k8s.Client, role string) {
	if role == "" {
		role = "worker"
	}

	mcpGVR := schema.GroupVersionResource{
		Group: "machineconfiguration.openshift.io", Version: "v1", Resource: "machineconfigpools",
	}

	// Wait up to 10 minutes for MCP to become Updated
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			return
		case <-ticker.C:
			mcp, err := client.Dynamic.Resource(mcpGVR).Get(ctx, role, metav1.GetOptions{})
			if err != nil {
				continue
			}

			conditions, found, _ := unstructured.NestedSlice(mcp.Object, "status", "conditions")
			if !found {
				continue
			}

			for _, cond := range conditions {
				condMap, ok := cond.(map[string]interface{})
				if !ok {
					continue
				}
				condType, _ := condMap["type"].(string)
				condStatus, _ := condMap["status"].(string)
				if condType == "Updated" && condStatus == "True" {
					return
				}
			}
		}
	}
}
