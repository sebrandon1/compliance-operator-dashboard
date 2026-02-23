package compliance

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateScan(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("creates ScanSettingBinding", func(t *testing.T) {
		client := newTestClient()

		opts := ScanOptions{
			Name:      "my-scan",
			Profile:   "ocp4-cis",
			Namespace: ns,
		}

		err := CreateScan(ctx, client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the SSB was created
		ssb, err := client.Dynamic.Resource(scanSettingBindingGVR).Namespace(ns).
			Get(ctx, "my-scan", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ScanSettingBinding not created: %v", err)
		}
		if ssb.GetName() != "my-scan" {
			t.Errorf("name = %q, want my-scan", ssb.GetName())
		}

		// Check profiles
		profiles, found, _ := unstructured.NestedSlice(ssb.Object, "profiles")
		if !found || len(profiles) == 0 {
			t.Fatal("expected profiles to be set")
		}
		firstProfile, ok := profiles[0].(map[string]any)
		if !ok {
			t.Fatal("expected profile to be a map")
		}
		profileName, _ := firstProfile["name"].(string)
		if profileName != "ocp4-cis" {
			t.Errorf("profile name = %q, want ocp4-cis", profileName)
		}

		// Check settingsRef
		settingsRef, found, _ := unstructured.NestedMap(ssb.Object, "settingsRef")
		if !found {
			t.Fatal("expected settingsRef to be set")
		}
		settingName, _ := settingsRef["name"].(string)
		if settingName != "default" {
			t.Errorf("settingsRef.name = %q, want default", settingName)
		}
	})

	t.Run("defaults namespace", func(t *testing.T) {
		client := newTestClient()

		opts := ScanOptions{
			Name:    "ns-test",
			Profile: "ocp4-cis",
			// Namespace intentionally empty
		}

		err := CreateScan(ctx, client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should use default namespace
		_, err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace("openshift-compliance").
			Get(ctx, "ns-test", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ScanSettingBinding not found in default namespace: %v", err)
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		err := CreateScan(ctx, nil, ScanOptions{Name: "test", Profile: "p"})
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func TestListProfiles(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("lists profiles", func(t *testing.T) {
		p1 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion":  "compliance.openshift.io/v1alpha1",
				"kind":        "Profile",
				"title":       "CIS Benchmark",
				"description": "CIS benchmark for OpenShift",
				"metadata": map[string]any{
					"name":      "ocp4-cis",
					"namespace": ns,
				},
			},
		}
		p2 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion":  "compliance.openshift.io/v1alpha1",
				"kind":        "Profile",
				"title":       "E8 Profile",
				"description": "Essential Eight",
				"metadata": map[string]any{
					"name":      "ocp4-e8",
					"namespace": ns,
				},
			},
		}

		client := newTestClient(p1, p2)

		profiles, err := ListProfiles(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 2 {
			t.Fatalf("got %d profiles, want 2", len(profiles))
		}

		// Check that profiles have the right fields
		found := false
		for _, p := range profiles {
			if p.Name == "ocp4-cis" {
				found = true
				if p.Title != "CIS Benchmark" {
					t.Errorf("Title = %q, want CIS Benchmark", p.Title)
				}
				if p.Description != "CIS benchmark for OpenShift" {
					t.Errorf("Description = %q, want CIS benchmark for OpenShift", p.Description)
				}
			}
		}
		if !found {
			t.Error("ocp4-cis profile not found in results")
		}
	})

	t.Run("empty when no profiles", func(t *testing.T) {
		client := newTestClient()

		profiles, err := ListProfiles(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("got %d profiles, want 0", len(profiles))
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := ListProfiles(ctx, nil, ns)
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func TestDeleteScan(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("deletes suite and SSB", func(t *testing.T) {
		suite := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "compliance.openshift.io/v1alpha1",
				"kind":       "ComplianceSuite",
				"metadata": map[string]any{
					"name":      "my-suite",
					"namespace": ns,
				},
			},
		}
		ssb := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "compliance.openshift.io/v1alpha1",
				"kind":       "ScanSettingBinding",
				"metadata": map[string]any{
					"name":      "my-suite",
					"namespace": ns,
				},
			},
		}

		client := newTestClient(suite, ssb)

		err := DeleteScan(ctx, client, ns, "my-suite")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify suite is deleted
		_, err = client.Dynamic.Resource(complianceSuiteGVR).Namespace(ns).
			Get(ctx, "my-suite", metav1.GetOptions{})
		if err == nil {
			t.Error("expected ComplianceSuite to be deleted")
		}

		// Verify SSB is deleted
		_, err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace(ns).
			Get(ctx, "my-suite", metav1.GetOptions{})
		if err == nil {
			t.Error("expected ScanSettingBinding to be deleted")
		}
	})

	t.Run("succeeds when resources not found", func(t *testing.T) {
		client := newTestClient()

		// Should not error when resources don't exist
		err := DeleteScan(ctx, client, ns, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		err := DeleteScan(ctx, nil, ns, "my-suite")
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func TestGetScanStatus(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("returns suite statuses", func(t *testing.T) {
		suite := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "compliance.openshift.io/v1alpha1",
				"kind":       "ComplianceSuite",
				"metadata": map[string]any{
					"name":              "test-suite",
					"namespace":         ns,
					"creationTimestamp": "2025-01-01T00:00:00Z",
				},
				"status": map[string]any{
					"phase":  "DONE",
					"result": "NON-COMPLIANT",
				},
			},
		}

		client := newTestClient(suite)

		statuses, err := GetScanStatus(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(statuses) != 1 {
			t.Fatalf("got %d statuses, want 1", len(statuses))
		}
		if statuses[0].Name != "test-suite" {
			t.Errorf("Name = %q, want test-suite", statuses[0].Name)
		}
		if statuses[0].Phase != "DONE" {
			t.Errorf("Phase = %q, want DONE", statuses[0].Phase)
		}
		if statuses[0].Result != "NON-COMPLIANT" {
			t.Errorf("Result = %q, want NON-COMPLIANT", statuses[0].Result)
		}
	})

	t.Run("empty when no suites", func(t *testing.T) {
		client := newTestClient()

		statuses, err := GetScanStatus(ctx, client, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(statuses) != 0 {
			t.Errorf("got %d statuses, want 0", len(statuses))
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := GetScanStatus(ctx, nil, ns)
		if err == nil {
			t.Error("expected error for nil client")
		}
	})
}

func TestCreateRecommendedScans(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	client := newTestClient()

	created, errs := CreateRecommendedScans(ctx, client, ns)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(created) != len(RecommendedProfiles) {
		t.Errorf("created %d scans, want %d", len(created), len(RecommendedProfiles))
	}

	// Verify each SSB was created
	for _, name := range created {
		_, err := client.Dynamic.Resource(scanSettingBindingGVR).Namespace(ns).
			Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("ScanSettingBinding %q not created: %v", name, err)
		}
	}
}
