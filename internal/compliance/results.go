package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	sigsyaml "sigs.k8s.io/yaml"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

// isCRDNotFound returns true if the error indicates the CRD is not installed.
func isCRDNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "the server could not find the requested resource") ||
		strings.Contains(msg, "no matches for kind")
}

var (
	complianceCheckResultGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancecheckresults",
	}
	complianceRemediationGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "complianceremediations",
	}
)

// GetComplianceResults collects all ComplianceCheckResults and groups them.
// Reimplements core/export-compliance-data.sh.
func GetComplianceResults(ctx context.Context, client *k8s.Client, namespace string) (*ComplianceData, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	// List all ComplianceCheckResults
	results, err := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		if isCRDNotFound(err) {
			return &ComplianceData{ScanDate: ScanTimestamp(), Summary: Summary{}}, nil
		}
		return nil, fmt.Errorf("listing ComplianceCheckResults: %w", err)
	}

	if len(results.Items) == 0 {
		return &ComplianceData{
			ScanDate: ScanTimestamp(),
			Summary:  Summary{},
		}, nil
	}

	data := &ComplianceData{
		ScanDate: ScanTimestamp(),
	}

	var (
		highFail, mediumFail, lowFail []CheckResult
		highPass, mediumPass, lowPass []CheckResult
		manualChecks                  []CheckResult
		totalPassing, totalFailing    int
		totalManual, totalSkipped     int
	)

	for _, item := range results.Items {
		cr := extractCheckResult(item)

		switch cr.Status {
		case CheckStatusPass:
			totalPassing++
			switch cr.Severity {
			case SeverityHigh:
				highPass = append(highPass, cr)
			case SeverityMedium:
				mediumPass = append(mediumPass, cr)
			case SeverityLow:
				lowPass = append(lowPass, cr)
			}

		case CheckStatusFail:
			totalFailing++
			switch cr.Severity {
			case SeverityHigh:
				highFail = append(highFail, cr)
			case SeverityMedium:
				mediumFail = append(mediumFail, cr)
			case SeverityLow:
				lowFail = append(lowFail, cr)
			}

		case CheckStatusManual:
			totalManual++
			manualChecks = append(manualChecks, cr)

		case CheckStatusSkip, CheckStatusNotApplicable:
			totalSkipped++
		}
	}

	data.Summary = Summary{
		TotalChecks: len(results.Items),
		Passing:     totalPassing,
		Failing:     totalFailing,
		Manual:      totalManual,
		Skipped:     totalSkipped,
	}

	data.Remediations = SeverityMap{
		High:   highFail,
		Medium: mediumFail,
		Low:    lowFail,
	}

	data.PassingChecks = SeverityMap{
		High:   highPass,
		Medium: mediumPass,
		Low:    lowPass,
	}

	data.ManualChecks = manualChecks

	return data, nil
}

// GetResultsSummary returns only the summary counts.
func GetResultsSummary(ctx context.Context, client *k8s.Client, namespace string) (*Summary, error) {
	data, err := GetComplianceResults(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	return &data.Summary, nil
}

// GetFilteredResults returns compliance results with optional filtering.
func GetFilteredResults(ctx context.Context, client *k8s.Client, namespace string, severity, status, search string) ([]CheckResult, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	results, err := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		if isCRDNotFound(err) {
			return []CheckResult{}, nil
		}
		return nil, fmt.Errorf("listing ComplianceCheckResults: %w", err)
	}

	var filtered []CheckResult
	for _, item := range results.Items {
		cr := extractCheckResult(item)

		// Apply severity filter
		if severity != "" && string(cr.Severity) != strings.ToLower(severity) {
			continue
		}

		// Apply status filter
		if status != "" && string(cr.Status) != strings.ToUpper(status) {
			continue
		}

		// Apply search filter
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(cr.Name), searchLower) &&
				!strings.Contains(strings.ToLower(cr.Description), searchLower) {
				continue
			}
		}

		filtered = append(filtered, cr)
	}

	return filtered, nil
}

// ListRemediations lists all ComplianceRemediations with severity information.
func ListRemediations(ctx context.Context, client *k8s.Client, namespace string) ([]RemediationInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	// Get remediations
	remediations, err := client.Dynamic.Resource(complianceRemediationGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		if isCRDNotFound(err) {
			return []RemediationInfo{}, nil
		}
		return nil, fmt.Errorf("listing ComplianceRemediations: %w", err)
	}

	// Build a name->severity map from ComplianceCheckResults
	severityMap := make(map[string]Severity)
	checkResults, err := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, cr := range checkResults.Items {
			name := cr.GetName()
			sev, _, _ := unstructured.NestedString(cr.Object, "severity")
			severityMap[name] = Severity(strings.ToLower(sev))
		}
	}

	var infos []RemediationInfo
	for _, rem := range remediations.Items {
		name := rem.GetName()

		// Extract kind from spec.current.object
		kind, _, _ := unstructured.NestedString(rem.Object, "spec", "current", "object", "kind")

		// Look up severity
		severity := severityMap[name]

		// Check if applied (handle both bool and string representations)
		applied := false
		if applyBool, found, err := unstructured.NestedBool(rem.Object, "spec", "apply"); err == nil && found {
			applied = applyBool
		} else if applyStr, found, err := unstructured.NestedString(rem.Object, "spec", "apply"); err == nil && found {
			applied = applyStr == "true"
		}

		// Determine if reboot is needed (MachineConfig changes reboot nodes)
		rebootNeeded := kind == "MachineConfig"

		// Determine role
		role := detectRole(name, rem)

		infos = append(infos, RemediationInfo{
			Name:         name,
			Kind:         kind,
			Severity:     severity,
			Applied:      applied,
			RebootNeeded: rebootNeeded,
			Role:         role,
		})
	}

	return infos, nil
}

// GetCheckResult fetches a single ComplianceCheckResult by name with full detail.
func GetCheckResult(ctx context.Context, client *k8s.Client, namespace, name string) (*CheckResultDetail, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	item, err := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting ComplianceCheckResult %s: %w", name, err)
	}

	status, _, _ := unstructured.NestedString(item.Object, "status")
	severity, _, _ := unstructured.NestedString(item.Object, "severity")
	description, _, _ := unstructured.NestedString(item.Object, "description")
	id, _, _ := unstructured.NestedString(item.Object, "id")
	instructions, _, _ := unstructured.NestedString(item.Object, "instructions")
	rationale, _, _ := unstructured.NestedString(item.Object, "rationale")

	// Extract scan association from labels
	labels := item.GetLabels()
	scanName := labels["compliance.openshift.io/scan-name"]
	suite := labels["compliance.openshift.io/suite"]

	detail := &CheckResultDetail{
		CheckResult: CheckResult{
			Name:        name,
			Check:       name,
			Status:      CheckStatus(strings.ToUpper(status)),
			Severity:    Severity(strings.ToLower(severity)),
			Description: description,
			ScanName:    scanName,
			Suite:       suite,
		},
		ID:           id,
		Instructions: instructions,
		Rationale:    rationale,
	}

	// Check for a matching remediation (exact match or prefix match)
	remediations, err := client.Dynamic.Resource(complianceRemediationGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, rem := range remediations.Items {
			remName := rem.GetName()
			if remName == name || strings.HasPrefix(remName, name+"-") {
				detail.HasRemediation = true
				detail.RemediationName = remName
				break
			}
		}
	}

	return detail, nil
}

// GetRemediation fetches a single ComplianceRemediation by name and returns its detail.
func GetRemediation(ctx context.Context, client *k8s.Client, namespace, name string) (*RemediationDetail, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	rem, err := client.Dynamic.Resource(complianceRemediationGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting ComplianceRemediation %s: %w", name, err)
	}

	// Extract kind from spec.current.object
	kind, _, _ := unstructured.NestedString(rem.Object, "spec", "current", "object", "kind")

	// Check if applied
	apply, _, _ := unstructured.NestedString(rem.Object, "spec", "apply")
	applied := apply == "true"

	// Determine reboot
	rebootNeeded := kind == "MachineConfig"

	// Determine role
	role := detectRole(name, *rem)

	// Look up severity from ComplianceCheckResult
	var severity Severity
	cr, crErr := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if crErr == nil {
		sev, _, _ := unstructured.NestedString(cr.Object, "severity")
		severity = Severity(strings.ToLower(sev))
	}

	// Extract the object YAML from spec.current.object
	obj, found, _ := unstructured.NestedMap(rem.Object, "spec", "current", "object")
	var objectYAML string
	if found && obj != nil {
		jsonBytes, err := json.Marshal(obj)
		if err == nil {
			yamlBytes, err := sigsyaml.JSONToYAML(jsonBytes)
			if err == nil {
				objectYAML = string(yamlBytes)
			}
		}
	}

	// Extract apiVersion from the inner object
	apiVersion, _, _ := unstructured.NestedString(rem.Object, "spec", "current", "object", "apiVersion")

	// Extract target namespace from the inner object
	objNamespace, _, _ := unstructured.NestedString(rem.Object, "spec", "current", "object", "metadata", "namespace")

	return &RemediationDetail{
		RemediationInfo: RemediationInfo{
			Name:         name,
			Kind:         kind,
			Severity:     severity,
			Applied:      applied,
			RebootNeeded: rebootNeeded,
			Role:         role,
		},
		ObjectYAML: objectYAML,
		APIVersion: apiVersion,
		Namespace:  objNamespace,
	}, nil
}

func extractCheckResult(item unstructured.Unstructured) CheckResult {
	name := item.GetName()

	// .status is a top-level string field in ComplianceCheckResult
	status, _, _ := unstructured.NestedString(item.Object, "status")
	severity, _, _ := unstructured.NestedString(item.Object, "severity")
	description, _, _ := unstructured.NestedString(item.Object, "description")

	// Extract scan association from labels
	labels := item.GetLabels()
	scanName := labels["compliance.openshift.io/scan-name"]
	suite := labels["compliance.openshift.io/suite"]

	return CheckResult{
		Name:        name,
		Check:       name,
		Status:      CheckStatus(strings.ToUpper(status)),
		Severity:    Severity(strings.ToLower(severity)),
		Description: description,
		ScanName:    scanName,
		Suite:       suite,
	}
}

func detectRole(name string, rem unstructured.Unstructured) string {
	// Check labels first
	labels := rem.GetLabels()
	if role, ok := labels["machineconfiguration.openshift.io/role"]; ok {
		return role
	}

	// Check name for role hints
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "master") {
		return "master"
	}
	if strings.Contains(nameLower, "worker") {
		return "worker"
	}

	// Check spec.current.object labels
	roleFromObj, _, _ := unstructured.NestedString(rem.Object,
		"spec", "current", "object", "metadata", "labels", "machineconfiguration.openshift.io/role")
	if roleFromObj != "" {
		return roleFromObj
	}

	return "worker" // Default
}
