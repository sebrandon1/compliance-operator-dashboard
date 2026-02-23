package compliance

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

// --- Tier 1: Pure function tests ---

func TestDefaultPeriodicScanOptions(t *testing.T) {
	opts := DefaultPeriodicScanOptions("test-ns")

	if opts.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want test-ns", opts.Namespace)
	}
	if opts.Schedule != "0 1 * * *" {
		t.Errorf("Schedule = %q, want '0 1 * * *'", opts.Schedule)
	}
	if len(opts.Profiles) != 3 {
		t.Errorf("got %d profiles, want 3", len(opts.Profiles))
	}
	if opts.StorageSize != "1Gi" {
		t.Errorf("StorageSize = %q, want 1Gi", opts.StorageSize)
	}
	if opts.Rotation != 3 {
		t.Errorf("Rotation = %d, want 3", opts.Rotation)
	}
	if len(opts.Roles) != 2 {
		t.Errorf("got %d roles, want 2", len(opts.Roles))
	}

	// Check expected profiles
	expectedProfiles := map[string]bool{
		"ocp4-cis":  true,
		"ocp4-e8":   true,
		"rhcos4-e8": true,
	}
	for _, p := range opts.Profiles {
		if !expectedProfiles[p] {
			t.Errorf("unexpected profile %q", p)
		}
	}

	// Check expected roles
	expectedRoles := map[string]bool{
		"worker": true,
		"master": true,
	}
	for _, r := range opts.Roles {
		if !expectedRoles[r] {
			t.Errorf("unexpected role %q", r)
		}
	}
}

func TestDefaultPeriodicScanOptions_EmptyNamespace(t *testing.T) {
	opts := DefaultPeriodicScanOptions("")
	if opts.Namespace != "" {
		t.Errorf("Namespace = %q, want empty", opts.Namespace)
	}
}

// --- Tier 2: Fake K8s client tests ---

// newTestClientWithPods creates a test client with both dynamic objects and typed pods.
func newTestClientWithPods(dynamicObjects []runtime.Object, pods []corev1.Pod) *k8s.Client {
	scheme := runtime.NewScheme()
	// Same GVK registrations as newTestClient
	gvks := []schema.GroupVersionKind{
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceCheckResultList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceRemediationList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceSuiteList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ComplianceScanList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ScanSettingBindingList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ScanSettingList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ProfileList"},
		{Group: "compliance.openshift.io", Version: "v1alpha1", Kind: "ProfileBundleList"},
		{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "SubscriptionList"},
		{Group: "operators.coreos.com", Version: "v1alpha1", Kind: "ClusterServiceVersionList"},
	}
	for _, gvk := range gvks {
		scheme.AddKnownTypeWithName(gvk, &unstructured.UnstructuredList{})
	}

	dynClient := dynamicfake.NewSimpleDynamicClient(scheme, dynamicObjects...)

	var kubeObjects []runtime.Object
	for i := range pods {
		kubeObjects = append(kubeObjects, &pods[i])
	}
	kubeClient := kubefake.NewClientset(kubeObjects...)

	return &k8s.Client{
		Clientset: kubeClient,
		Dynamic:   dynClient,
	}
}

func TestGetStatus_Installed(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	sub := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]any{
				"name":      subscriptionName,
				"namespace": ns,
			},
			"status": map[string]any{
				"installedCSV": "compliance-operator.v1.5.0",
			},
		},
	}

	csv := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "ClusterServiceVersion",
			"metadata": map[string]any{
				"name":      "compliance-operator.v1.5.0",
				"namespace": ns,
			},
			"status": map[string]any{
				"phase": "Succeeded",
			},
		},
	}

	bundle := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ProfileBundle",
			"metadata": map[string]any{
				"name":      "ocp4",
				"namespace": ns,
			},
			"status": map[string]any{
				"dataStreamStatus": "VALID",
			},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "compliance-operator-abc",
				Namespace: ns,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		},
	}

	client := newTestClientWithPods([]runtime.Object{sub, csv, bundle}, pods)

	status, err := GetStatus(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Installed {
		t.Error("expected Installed=true")
	}
	if status.Version != "compliance-operator.v1.5.0" {
		t.Errorf("Version = %q, want compliance-operator.v1.5.0", status.Version)
	}
	if status.CSVPhase != "Succeeded" {
		t.Errorf("CSVPhase = %q, want Succeeded", status.CSVPhase)
	}
	if len(status.Pods) != 1 {
		t.Fatalf("got %d pods, want 1", len(status.Pods))
	}
	if !status.Pods[0].Ready {
		t.Error("expected pod to be ready")
	}
	if status.Pods[0].Name != "compliance-operator-abc" {
		t.Errorf("Pod name = %q, want compliance-operator-abc", status.Pods[0].Name)
	}
	if len(status.ProfileBundles) != 1 {
		t.Fatalf("got %d profile bundles, want 1", len(status.ProfileBundles))
	}
	if status.ProfileBundles[0].Name != "ocp4" {
		t.Errorf("ProfileBundle name = %q, want ocp4", status.ProfileBundles[0].Name)
	}
	if status.ProfileBundles[0].DataStreamStatus != "VALID" {
		t.Errorf("DataStreamStatus = %q, want VALID", status.ProfileBundles[0].DataStreamStatus)
	}
}

func TestGetStatus_NotInstalled(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	client := newTestClient()

	status, err := GetStatus(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Installed {
		t.Error("expected Installed=false when no subscription")
	}
}

func TestGetStatus_NilClient(t *testing.T) {
	status, err := GetStatus(context.Background(), nil, "ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Installed {
		t.Error("expected Installed=false for nil client")
	}
}

func TestGetStatus_SubscriptionWithoutCSV(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	sub := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]any{
				"name":      subscriptionName,
				"namespace": ns,
			},
			"status": map[string]any{},
		},
	}

	client := newTestClient(sub)

	status, err := GetStatus(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No installedCSV means not fully installed
	if status.Installed {
		t.Error("expected Installed=false when no installedCSV")
	}
}

func TestGetStatus_PodNotReady(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	sub := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]any{
				"name":      subscriptionName,
				"namespace": ns,
			},
			"status": map[string]any{
				"installedCSV": "compliance-operator.v1.5.0",
			},
		},
	}

	csv := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "ClusterServiceVersion",
			"metadata": map[string]any{
				"name":      "compliance-operator.v1.5.0",
				"namespace": ns,
			},
			"status": map[string]any{
				"phase": "Succeeded",
			},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "compliance-operator-xyz",
				Namespace: ns,
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionFalse},
				},
			},
		},
	}

	client := newTestClientWithPods([]runtime.Object{sub, csv}, pods)

	status, err := GetStatus(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.Installed {
		t.Error("expected Installed=true")
	}
	if len(status.Pods) != 1 {
		t.Fatalf("got %d pods, want 1", len(status.Pods))
	}
	if status.Pods[0].Ready {
		t.Error("expected pod to not be ready")
	}
	if status.Pods[0].Phase != "Pending" {
		t.Errorf("Phase = %q, want Pending", status.Pods[0].Phase)
	}
}

func TestGetStatus_MultipleProfileBundles(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	sub := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]any{
				"name":      subscriptionName,
				"namespace": ns,
			},
			"status": map[string]any{
				"installedCSV": "compliance-operator.v1.5.0",
			},
		},
	}

	csv := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "ClusterServiceVersion",
			"metadata": map[string]any{
				"name":      "compliance-operator.v1.5.0",
				"namespace": ns,
			},
			"status": map[string]any{
				"phase": "Succeeded",
			},
		},
	}

	bundle1 := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ProfileBundle",
			"metadata": map[string]any{
				"name":      "ocp4",
				"namespace": ns,
			},
			"status": map[string]any{
				"dataStreamStatus": "VALID",
			},
		},
	}

	bundle2 := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ProfileBundle",
			"metadata": map[string]any{
				"name":      "rhcos4",
				"namespace": ns,
			},
			"status": map[string]any{
				"dataStreamStatus": "VALID",
			},
		},
	}

	client := newTestClient(sub, csv, bundle1, bundle2)

	status, err := GetStatus(ctx, client, ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(status.ProfileBundles) != 2 {
		t.Errorf("got %d profile bundles, want 2", len(status.ProfileBundles))
	}
}
