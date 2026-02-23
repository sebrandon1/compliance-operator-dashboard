package compliance

import (
	"context"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

func newTestClient(objects ...runtime.Object) *k8s.Client {
	scheme := runtime.NewScheme()
	// Register GVRs so the fake client can handle them.
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceCheckResultList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceRemediationList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceSuiteList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceScanList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ScanSettingBindingList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ScanSettingList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ProfileList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ProfileBundleList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "SubscriptionList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "ClusterServiceVersionList"},
		&unstructured.UnstructuredList{},
	)
	// Register cluster-scoped resource lists used by remediation tests.
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "machineconfiguration.openshift.io", Version: "v1", Kind: "MachineConfigList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMapList"},
		&unstructured.UnstructuredList{},
	)

	dynClient := dynamicfake.NewSimpleDynamicClient(scheme, objects...)
	kubeClient := kubefake.NewClientset()
	return &k8s.Client{
		Clientset: kubeClient,
		Dynamic:   dynClient,
	}
}

func newCheckResult(name, namespace, status, severity, description, scanName, suite string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion":  "compliance.openshift.io/v1alpha1",
			"kind":        "ComplianceCheckResult",
			"status":      status,
			"severity":    severity,
			"description": description,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	labels := map[string]string{}
	if scanName != "" {
		labels["compliance.openshift.io/scan-name"] = scanName
	}
	if suite != "" {
		labels["compliance.openshift.io/suite"] = suite
	}
	if len(labels) > 0 {
		obj.SetLabels(labels)
	}
	return obj
}

func newRemediation(name, namespace string, fields map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ComplianceRemediation",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	for k, v := range fields {
		obj.Object[k] = v
	}
	return obj
}

// --- Tier 1: Pure function tests ---

func TestExtractCheckResult(t *testing.T) {
	tests := []struct {
		name     string
		item     unstructured.Unstructured
		expected CheckResult
	}{
		{
			name: "full fields",
			item: *newCheckResult("check-1", "ns", "PASS", "high", "desc1", "scan-a", "suite-a"),
			expected: CheckResult{
				Name:        "check-1",
				Check:       "check-1",
				Status:      CheckStatusPass,
				Severity:    SeverityHigh,
				Description: "desc1",
				ScanName:    "scan-a",
				Suite:       "suite-a",
			},
		},
		{
			name: "missing severity",
			item: *newCheckResult("check-2", "ns", "FAIL", "", "desc2", "", ""),
			expected: CheckResult{
				Name:        "check-2",
				Check:       "check-2",
				Status:      CheckStatusFail,
				Severity:    Severity(""),
				Description: "desc2",
			},
		},
		{
			name: "missing status",
			item: *newCheckResult("check-3", "ns", "", "medium", "", "scan-b", ""),
			expected: CheckResult{
				Name:     "check-3",
				Check:    "check-3",
				Status:   CheckStatus(""),
				Severity: SeverityMedium,
				ScanName: "scan-b",
			},
		},
		{
			name: "lowercase status gets uppercased",
			item: *newCheckResult("check-4", "ns", "fail", "low", "desc4", "", ""),
			expected: CheckResult{
				Name:        "check-4",
				Check:       "check-4",
				Status:      CheckStatusFail,
				Severity:    SeverityLow,
				Description: "desc4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCheckResult(tt.item)
			if got.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.expected.Name)
			}
			if got.Check != tt.expected.Check {
				t.Errorf("Check = %q, want %q", got.Check, tt.expected.Check)
			}
			if got.Status != tt.expected.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.expected.Status)
			}
			if got.Severity != tt.expected.Severity {
				t.Errorf("Severity = %q, want %q", got.Severity, tt.expected.Severity)
			}
			if got.Description != tt.expected.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.expected.Description)
			}
			if got.ScanName != tt.expected.ScanName {
				t.Errorf("ScanName = %q, want %q", got.ScanName, tt.expected.ScanName)
			}
			if got.Suite != tt.expected.Suite {
				t.Errorf("Suite = %q, want %q", got.Suite, tt.expected.Suite)
			}
		})
	}
}

func TestIsCRDNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "server could not find resource",
			err:  fmt.Errorf("the server could not find the requested resource"),
			want: true,
		},
		{
			name: "no matches for kind",
			err:  fmt.Errorf("no matches for kind \"ComplianceCheckResult\" in version \"compliance.openshift.io/v1alpha1\""),
			want: true,
		},
		{
			name: "wrapped server could not find",
			err:  fmt.Errorf("listing: %w", fmt.Errorf("the server could not find the requested resource")),
			want: true,
		},
		{
			name: "unrelated error",
			err:  fmt.Errorf("connection refused"),
			want: false,
		},
		{
			name: "not found but different message",
			err:  fmt.Errorf("resource not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCRDNotFound(tt.err)
			if got != tt.want {
				t.Errorf("isCRDNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestDetectRole(t *testing.T) {
	tests := []struct {
		name     string
		remName  string
		rem      unstructured.Unstructured
		expected string
	}{
		{
			name:    "role from label",
			remName: "some-remediation",
			rem: func() unstructured.Unstructured {
				obj := unstructured.Unstructured{Object: map[string]interface{}{}}
				obj.SetLabels(map[string]string{
					"machineconfiguration.openshift.io/role": "master",
				})
				return obj
			}(),
			expected: "master",
		},
		{
			name:     "master from name",
			remName:  "ocp4-cis-master-audit-rules",
			rem:      unstructured.Unstructured{Object: map[string]interface{}{}},
			expected: "master",
		},
		{
			name:     "worker from name",
			remName:  "rhcos4-worker-sshd-config",
			rem:      unstructured.Unstructured{Object: map[string]interface{}{}},
			expected: "worker",
		},
		{
			name:    "role from nested object labels",
			remName: "some-remediation",
			rem: unstructured.Unstructured{Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"current": map[string]interface{}{
						"object": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"machineconfiguration.openshift.io/role": "infra",
								},
							},
						},
					},
				},
			}},
			expected: "infra",
		},
		{
			name:     "fallback to worker",
			remName:  "some-generic-remediation",
			rem:      unstructured.Unstructured{Object: map[string]interface{}{}},
			expected: "worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectRole(tt.remName, tt.rem)
			if got != tt.expected {
				t.Errorf("detectRole(%q, ...) = %q, want %q", tt.remName, got, tt.expected)
			}
		})
	}
}

// --- Tier 2: Fake K8s client tests ---

func TestGetFilteredResults(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr1 := newCheckResult("check-high-fail", ns, "FAIL", "high", "high severity failure", "scan-1", "suite-1")
	cr2 := newCheckResult("check-medium-pass", ns, "PASS", "medium", "medium severity pass", "scan-1", "suite-1")
	cr3 := newCheckResult("check-low-fail", ns, "FAIL", "low", "low severity issue", "scan-1", "suite-1")
	cr4 := newCheckResult("check-high-pass", ns, "PASS", "high", "high severity pass", "scan-1", "suite-1")

	client := newTestClient(cr1, cr2, cr3, cr4)

	t.Run("no filters returns all", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, ns, "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 4 {
			t.Errorf("got %d results, want 4", len(results))
		}
	})

	t.Run("filter by severity", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, ns, "high", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
		for _, r := range results {
			if r.Severity != SeverityHigh {
				t.Errorf("expected severity high, got %q", r.Severity)
			}
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, ns, "", "FAIL", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}
		for _, r := range results {
			if r.Status != CheckStatusFail {
				t.Errorf("expected status FAIL, got %q", r.Status)
			}
		}
	})

	t.Run("filter by search", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, ns, "", "", "issue")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Name != "check-low-fail" {
			t.Errorf("expected check-low-fail, got %q", results[0].Name)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, ns, "high", "FAIL", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
		if len(results) > 0 && results[0].Name != "check-high-fail" {
			t.Errorf("expected check-high-fail, got %q", results[0].Name)
		}
	})

	t.Run("empty namespace returns empty", func(t *testing.T) {
		results, err := GetFilteredResults(ctx, client, "nonexistent", "", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := GetFilteredResults(ctx, nil, ns, "", "", "")
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func TestListRemediations(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	// Remediation with spec.apply as bool true
	rem1 := newRemediation("rem-bool-true", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": true,
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind": "MachineConfig",
				},
			},
		},
	})

	// Remediation with spec.apply as string "true"
	rem2 := newRemediation("rem-string-true", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": "true",
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind": "ConfigMap",
				},
			},
		},
	})

	// Remediation with spec.apply as bool false
	rem3 := newRemediation("rem-not-applied", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": false,
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind": "Secret",
				},
			},
		},
	})

	// Matching check results for severity lookup
	cr1 := newCheckResult("rem-bool-true", ns, "FAIL", "high", "", "", "")
	cr2 := newCheckResult("rem-string-true", ns, "FAIL", "medium", "", "", "")

	client := newTestClient(rem1, rem2, rem3, cr1, cr2)

	t.Run("lists all remediations", func(t *testing.T) {
		infos, err := ListRemediations(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(infos) != 3 {
			t.Fatalf("got %d remediations, want 3", len(infos))
		}
	})

	t.Run("bool apply detected", func(t *testing.T) {
		infos, err := ListRemediations(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := findRemediation(infos, "rem-bool-true")
		if found == nil {
			t.Fatal("rem-bool-true not found")
		}
		if !found.Applied {
			t.Error("expected Applied=true for bool true")
		}
		if found.Kind != "MachineConfig" {
			t.Errorf("Kind = %q, want MachineConfig", found.Kind)
		}
		if !found.RebootNeeded {
			t.Error("expected RebootNeeded=true for MachineConfig")
		}
	})

	t.Run("string apply detected", func(t *testing.T) {
		infos, err := ListRemediations(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := findRemediation(infos, "rem-string-true")
		if found == nil {
			t.Fatal("rem-string-true not found")
		}
		if !found.Applied {
			t.Error("expected Applied=true for string \"true\"")
		}
	})

	t.Run("not applied detected", func(t *testing.T) {
		infos, err := ListRemediations(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := findRemediation(infos, "rem-not-applied")
		if found == nil {
			t.Fatal("rem-not-applied not found")
		}
		if found.Applied {
			t.Error("expected Applied=false")
		}
	})

	t.Run("severity from check results", func(t *testing.T) {
		infos, err := ListRemediations(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		found := findRemediation(infos, "rem-bool-true")
		if found == nil {
			t.Fatal("rem-bool-true not found")
		}
		if found.Severity != SeverityHigh {
			t.Errorf("Severity = %q, want high", found.Severity)
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := ListRemediations(ctx, nil, ns)
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func findRemediation(infos []RemediationInfo, name string) *RemediationInfo {
	for i := range infos {
		if infos[i].Name == name {
			return &infos[i]
		}
	}
	return nil
}

func TestGetFilteredResults_CRDNotFound(t *testing.T) {
	ctx := context.Background()
	// Client with no objects registered - the fake dynamic client will return
	// an empty list for the namespace, which is the expected empty-results behavior.
	client := newTestClient()
	results, err := GetFilteredResults(ctx, client, "openshift-compliance", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0 for empty namespace", len(results))
	}
}

func TestListRemediations_Empty(t *testing.T) {
	ctx := context.Background()
	client := newTestClient()
	infos, err := ListRemediations(ctx, client, "openshift-compliance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("got %d remediations, want 0", len(infos))
	}
}

func TestGetComplianceResults(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr1 := newCheckResult("pass-high", ns, "PASS", "high", "d1", "scan", "suite")
	cr2 := newCheckResult("fail-medium", ns, "FAIL", "medium", "d2", "scan", "suite")
	cr3 := newCheckResult("manual-low", ns, "MANUAL", "low", "d3", "scan", "suite")
	cr4 := newCheckResult("skip-high", ns, "SKIP", "high", "d4", "scan", "suite")

	client := newTestClient(cr1, cr2, cr3, cr4)

	data, err := GetComplianceResults(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Summary.TotalChecks != 4 {
		t.Errorf("TotalChecks = %d, want 4", data.Summary.TotalChecks)
	}
	if data.Summary.Passing != 1 {
		t.Errorf("Passing = %d, want 1", data.Summary.Passing)
	}
	if data.Summary.Failing != 1 {
		t.Errorf("Failing = %d, want 1", data.Summary.Failing)
	}
	if data.Summary.Manual != 1 {
		t.Errorf("Manual = %d, want 1", data.Summary.Manual)
	}
	if data.Summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", data.Summary.Skipped)
	}
	if data.ScanDate == "" {
		t.Error("ScanDate should not be empty")
	}
}

func TestGetComplianceResults_NilClient(t *testing.T) {
	_, err := GetComplianceResults(context.Background(), nil, "ns")
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestGetComplianceResults_Empty(t *testing.T) {
	ctx := context.Background()
	client := newTestClient()
	data, err := GetComplianceResults(ctx, client, "openshift-compliance")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Summary.TotalChecks != 0 {
		t.Errorf("TotalChecks = %d, want 0", data.Summary.TotalChecks)
	}
}

func TestGetCheckResult_Found(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion":   "compliance.openshift.io/v1alpha1",
			"kind":         "ComplianceCheckResult",
			"status":       "FAIL",
			"severity":     "high",
			"description":  "test desc",
			"id":           "xccdf_rule_123",
			"instructions": "do this",
			"rationale":    "because",
			"metadata": map[string]interface{}{
				"name":      "my-check",
				"namespace": ns,
				"labels": map[string]interface{}{
					"compliance.openshift.io/scan-name": "scan-a",
					"compliance.openshift.io/suite":     "suite-a",
				},
			},
		},
	}

	client := newTestClient(cr)

	detail, err := GetCheckResult(ctx, client, ns, "my-check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "my-check" {
		t.Errorf("Name = %q, want my-check", detail.Name)
	}
	if detail.Status != CheckStatusFail {
		t.Errorf("Status = %q, want FAIL", detail.Status)
	}
	if detail.Severity != SeverityHigh {
		t.Errorf("Severity = %q, want high", detail.Severity)
	}
	if detail.ID != "xccdf_rule_123" {
		t.Errorf("ID = %q, want xccdf_rule_123", detail.ID)
	}
	if detail.Instructions != "do this" {
		t.Errorf("Instructions = %q, want 'do this'", detail.Instructions)
	}
	if detail.Rationale != "because" {
		t.Errorf("Rationale = %q, want 'because'", detail.Rationale)
	}
}

func TestGetCheckResult_NotFound(t *testing.T) {
	ctx := context.Background()
	client := newTestClient()

	_, err := GetCheckResult(ctx, client, "openshift-compliance", "nonexistent")
	if err == nil {
		t.Error("expected error for missing check result")
	}
}

func TestGetCheckResult_WithRemediation(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr := newCheckResult("my-check", ns, "FAIL", "high", "desc", "scan-a", "suite-a")
	rem := newRemediation("my-check", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": false,
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind": "ConfigMap",
				},
			},
		},
	})

	client := newTestClient(cr, rem)

	detail, err := GetCheckResult(ctx, client, ns, "my-check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detail.HasRemediation {
		t.Error("expected HasRemediation=true")
	}
	if detail.RemediationName != "my-check" {
		t.Errorf("RemediationName = %q, want my-check", detail.RemediationName)
	}
}

func TestGetRemediation_Found(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	rem := newRemediation("my-rem", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": "true",
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind":       "ConfigMap",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name":      "my-cm",
						"namespace": "target-ns",
					},
				},
			},
		},
	})
	cr := newCheckResult("my-rem", ns, "FAIL", "high", "desc", "", "")

	client := newTestClient(rem, cr)

	detail, err := GetRemediation(ctx, client, ns, "my-rem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "my-rem" {
		t.Errorf("Name = %q, want my-rem", detail.Name)
	}
	if detail.Kind != "ConfigMap" {
		t.Errorf("Kind = %q, want ConfigMap", detail.Kind)
	}
	if !detail.Applied {
		t.Error("expected Applied=true")
	}
	if detail.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want v1", detail.APIVersion)
	}
	if detail.Namespace != "target-ns" {
		t.Errorf("Namespace = %q, want target-ns", detail.Namespace)
	}
	if detail.Severity != SeverityHigh {
		t.Errorf("Severity = %q, want high", detail.Severity)
	}
	if detail.ObjectYAML == "" {
		t.Error("expected non-empty ObjectYAML")
	}
}

func TestGetResultsSummary(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr1 := newCheckResult("p1", ns, "PASS", "high", "", "", "")
	cr2 := newCheckResult("f1", ns, "FAIL", "medium", "", "", "")

	client := newTestClient(cr1, cr2)

	summary, err := GetResultsSummary(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.TotalChecks != 2 {
		t.Errorf("TotalChecks = %d, want 2", summary.TotalChecks)
	}
	if summary.Passing != 1 {
		t.Errorf("Passing = %d, want 1", summary.Passing)
	}
	if summary.Failing != 1 {
		t.Errorf("Failing = %d, want 1", summary.Failing)
	}
}

func TestGetResultsSummary_NilClient(t *testing.T) {
	_, err := GetResultsSummary(context.Background(), nil, "ns")
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestListRemediations_DetectRole(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	// Remediation with master in the name
	rem := newRemediation("ocp4-cis-master-audit", ns, map[string]interface{}{
		"spec": map[string]interface{}{
			"apply": false,
			"current": map[string]interface{}{
				"object": map[string]interface{}{
					"kind": "ConfigMap",
				},
			},
		},
	})

	client := newTestClient(rem)

	infos, err := ListRemediations(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("got %d remediations, want 1", len(infos))
	}
	if infos[0].Role != "master" {
		t.Errorf("Role = %q, want master", infos[0].Role)
	}
}

func TestGetFilteredResults_SearchByName(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr := newCheckResult("ocp4-cis-audit-rules", ns, "FAIL", "high", "some description", "", "")
	client := newTestClient(cr)

	results, err := GetFilteredResults(ctx, client, ns, "", "", "audit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (search by name)", len(results))
	}
}

func TestGetFilteredResults_CaseInsensitiveSearch(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr := newCheckResult("check-1", ns, "FAIL", "high", "Kernel Module Loading", "", "")
	client := newTestClient(cr)

	results, err := GetFilteredResults(ctx, client, ns, "", "", "kernel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (case-insensitive search)", len(results))
	}
}

func TestGetFilteredResults_CaseInsensitiveSeverity(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	cr := newCheckResult("check-1", ns, "FAIL", "high", "desc", "", "")
	client := newTestClient(cr)

	// Severity filter uses ToLower, so "HIGH" should match stored "high"
	results, err := GetFilteredResults(ctx, client, ns, "HIGH", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (case-insensitive severity)", len(results))
	}
}

func TestScanTimestamp(t *testing.T) {
	ts := ScanTimestamp()
	if ts == "" {
		t.Error("ScanTimestamp should not return empty string")
	}
}

func TestNewService(t *testing.T) {
	client := newTestClient()
	svc := NewService(client, "test-ns", "v1.0.0")
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
	if svc.K8sClient() != client {
		t.Error("K8sClient() should return the provided client")
	}
}

func TestServiceK8sClient_NilService(t *testing.T) {
	var svc *Service
	if svc.K8sClient() != nil {
		t.Error("K8sClient() on nil Service should return nil")
	}
}

// Verify that fake client properly stores objects so List can find them in the right namespace.
func TestFakeClientListByNamespace(t *testing.T) {
	ctx := context.Background()
	ns := "test-ns"

	cr := newCheckResult("check-1", ns, "PASS", "high", "desc", "", "")
	client := newTestClient(cr)

	results, err := client.Dynamic.Resource(complianceCheckResultGVR).Namespace(ns).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Items) != 1 {
		t.Errorf("got %d items, want 1", len(results.Items))
	}

	// Different namespace should return 0
	results, err = client.Dynamic.Resource(complianceCheckResultGVR).Namespace("other-ns").
		List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Items) != 0 {
		t.Errorf("got %d items in other-ns, want 0", len(results.Items))
	}
}
