package compliance

import (
	"context"
	"fmt"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

var (
	scanSettingGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "scansettings",
	}
	scanSettingBindingGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "scansettingbindings",
	}
	complianceSuiteGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancesuites",
	}
	complianceScanGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancescans",
	}
	profileGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "profiles",
	}
)

// CreateScan creates a one-off compliance scan using a ScanSettingBinding.
// Reimplements core/create-scan.sh.
func CreateScan(ctx context.Context, client *k8s.Client, opts ScanOptions) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "openshift-compliance"
	}

	ssb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ScanSettingBinding",
			"metadata": map[string]interface{}{
				"name":      opts.Name,
				"namespace": namespace,
			},
			"profiles": []interface{}{
				map[string]interface{}{
					"apiGroup": "compliance.openshift.io/v1alpha1",
					"kind":     "Profile",
					"name":     opts.Profile,
				},
			},
			"settingsRef": map[string]interface{}{
				"apiGroup": "compliance.openshift.io/v1alpha1",
				"kind":     "ScanSetting",
				"name":     "default",
			},
		},
	}

	_, err := client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
		Create(ctx, ssb, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			// Update instead
			_, err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
				Update(ctx, ssb, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("updating ScanSettingBinding: %w", err)
			}
			return nil
		}
		return fmt.Errorf("creating ScanSettingBinding: %w", err)
	}

	return nil
}

// CreatePeriodicScan creates a periodic scan configuration with ScanSetting and ScanSettingBindings.
// Reimplements core/apply-periodic-scan.sh.
func CreatePeriodicScan(ctx context.Context, client *k8s.Client, opts PeriodicScanOptions) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "openshift-compliance"
	}

	// Build ScanSetting spec
	scanSettingSpec := map[string]interface{}{
		"schedule": opts.Schedule,
	}

	// Add roles
	roles := opts.Roles
	if len(roles) == 0 {
		roles = []string{"worker", "master"}
	}
	rolesSlice := make([]interface{}, len(roles))
	for i, r := range roles {
		rolesSlice[i] = r
	}
	scanSettingSpec["roles"] = rolesSlice

	// Add rawResultStorage if StorageClassName is provided
	if opts.StorageClassName != "" {
		storageSize := opts.StorageSize
		if storageSize == "" {
			storageSize = "1Gi"
		}
		rotation := opts.Rotation
		if rotation == 0 {
			rotation = 3
		}
		scanSettingSpec["rawResultStorage"] = map[string]interface{}{
			"storageClassName": opts.StorageClassName,
			"size":             storageSize,
			"rotation":         int64(rotation),
			"tolerations": []interface{}{
				map[string]interface{}{
					"key": "node-role.kubernetes.io/master", "operator": "Exists", "effect": "NoSchedule",
				},
				map[string]interface{}{
					"key": "node.kubernetes.io/not-ready", "operator": "Exists", "effect": "NoExecute",
					"tolerationSeconds": int64(300),
				},
				map[string]interface{}{
					"key": "node.kubernetes.io/unreachable", "operator": "Exists", "effect": "NoExecute",
					"tolerationSeconds": int64(300),
				},
				map[string]interface{}{
					"key": "node.kubernetes.io/memory-pressure", "operator": "Exists", "effect": "NoSchedule",
				},
			},
		}
	}

	// Create ScanSetting
	ss := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ScanSetting",
			"metadata": map[string]interface{}{
				"name":      "periodic-setting",
				"namespace": namespace,
			},
		},
	}
	for k, v := range scanSettingSpec {
		ss.Object[k] = v
	}

	_, err := client.Dynamic.Resource(scanSettingGVR).Namespace(namespace).
		Create(ctx, ss, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			ss.SetResourceVersion("")
			existing, getErr := client.Dynamic.Resource(scanSettingGVR).Namespace(namespace).
				Get(ctx, "periodic-setting", metav1.GetOptions{})
			if getErr == nil {
				ss.SetResourceVersion(existing.GetResourceVersion())
			}
			_, err = client.Dynamic.Resource(scanSettingGVR).Namespace(namespace).
				Update(ctx, ss, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("updating ScanSetting: %w", err)
			}
		} else {
			return fmt.Errorf("creating ScanSetting: %w", err)
		}
	}

	// Create ScanSettingBindings for each profile group
	// E8 profiles binding
	e8Profiles := []interface{}{}
	cisProfiles := []interface{}{}

	for _, profile := range opts.Profiles {
		entry := map[string]interface{}{
			"name":     profile,
			"kind":     "Profile",
			"apiGroup": "compliance.openshift.io/v1alpha1",
		}
		if strings.Contains(profile, "e8") {
			e8Profiles = append(e8Profiles, entry)
		} else if strings.Contains(profile, "cis") {
			cisProfiles = append(cisProfiles, entry)
		} else {
			// Default to CIS binding
			cisProfiles = append(cisProfiles, entry)
		}
	}

	settingsRef := map[string]interface{}{
		"name":     "periodic-setting",
		"kind":     "ScanSetting",
		"apiGroup": "compliance.openshift.io/v1alpha1",
	}

	if len(e8Profiles) > 0 {
		if err := createOrUpdateSSB(ctx, client, namespace, "periodic-e8", e8Profiles, settingsRef); err != nil {
			return fmt.Errorf("creating E8 ScanSettingBinding: %w", err)
		}
	}

	if len(cisProfiles) > 0 {
		if err := createOrUpdateSSB(ctx, client, namespace, "cis-scan", cisProfiles, settingsRef); err != nil {
			return fmt.Errorf("creating CIS ScanSettingBinding: %w", err)
		}
	}

	return nil
}

func createOrUpdateSSB(ctx context.Context, client *k8s.Client, namespace, name string, profiles []interface{}, settingsRef map[string]interface{}) error {
	ssb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "compliance.openshift.io/v1alpha1",
			"kind":       "ScanSettingBinding",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"profiles":    profiles,
			"settingsRef": settingsRef,
		},
	}

	_, err := client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
		Create(ctx, ssb, metav1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			existing, getErr := client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
				Get(ctx, name, metav1.GetOptions{})
			if getErr == nil {
				ssb.SetResourceVersion(existing.GetResourceVersion())
			}
			_, err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
				Update(ctx, ssb, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

// GetScanStatus lists all ComplianceSuites and ComplianceScans with their phase.
func GetScanStatus(ctx context.Context, client *k8s.Client, namespace string) ([]SuiteStatus, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	var statuses []SuiteStatus

	// List ComplianceSuites
	suites, err := client.Dynamic.Resource(complianceSuiteGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		if isCRDNotFound(err) {
			return []SuiteStatus{}, nil
		}
		return nil, fmt.Errorf("listing ComplianceSuites: %w", err)
	}

	// Build a map of ComplianceScan details
	scanDetails := make(map[string]ScanStatus)
	scans, scanErr := client.Dynamic.Resource(complianceScanGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if scanErr == nil {
		for _, scan := range scans.Items {
			name := scan.GetName()
			phase, _, _ := unstructured.NestedString(scan.Object, "status", "phase")
			result, _, _ := unstructured.NestedString(scan.Object, "status", "result")
			profile, _, _ := unstructured.NestedString(scan.Object, "spec", "profile")
			scanType, _, _ := unstructured.NestedString(scan.Object, "spec", "scanType")
			contentImage, _, _ := unstructured.NestedString(scan.Object, "spec", "contentImage")
			startTS, _, _ := unstructured.NestedString(scan.Object, "status", "startTimestamp")
			endTS, _, _ := unstructured.NestedString(scan.Object, "status", "endTimestamp")
			warnings, _, _ := unstructured.NestedString(scan.Object, "status", "warnings")

			scanDetails[name] = ScanStatus{
				Name:           name,
				Phase:          phase,
				Result:         result,
				Profile:        profile,
				ScanType:       scanType,
				ContentImage:   contentImage,
				StartTimestamp: startTS,
				EndTimestamp:   endTS,
				Warnings:       warnings,
			}
		}
	}

	for _, suite := range suites.Items {
		phase, _, _ := unstructured.NestedString(suite.Object, "status", "phase")
		result, _, _ := unstructured.NestedString(suite.Object, "status", "result")

		ss := SuiteStatus{
			Name:      suite.GetName(),
			Phase:     phase,
			Result:    result,
			CreatedAt: suite.GetCreationTimestamp().Format("2006-01-02T15:04:05Z"),
		}

		// Extract conditions
		conditions, _, _ := unstructured.NestedSlice(suite.Object, "status", "conditions")
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := condMap["type"].(string)
			condStatus, _ := condMap["status"].(string)
			reason, _ := condMap["reason"].(string)
			lastTransition, _ := condMap["lastTransitionTime"].(string)

			ss.Conditions = append(ss.Conditions, Condition{
				Type:               condType,
				Status:             condStatus,
				Reason:             reason,
				LastTransitionTime: lastTransition,
			})
		}

		// Get associated scans with full detail
		scanStatuses, _, _ := unstructured.NestedSlice(suite.Object, "status", "scanStatuses")
		for _, scanStatus := range scanStatuses {
			scanMap, ok := scanStatus.(map[string]interface{})
			if !ok {
				continue
			}
			scanName, _ := scanMap["name"].(string)

			// Use full scan details if available, otherwise fallback
			if detail, found := scanDetails[scanName]; found {
				ss.Scans = append(ss.Scans, detail)
			} else {
				scanPhase, _ := scanMap["phase"].(string)
				ss.Scans = append(ss.Scans, ScanStatus{
					Name:  scanName,
					Phase: scanPhase,
				})
			}
		}

		statuses = append(statuses, ss)
	}

	return statuses, nil
}

// RecommendedProfiles is the set of profiles that provide broad compliance
// coverage without redundancy: CIS, NIST 800-53 Moderate (platform + node),
// and PCI-DSS.
var RecommendedProfiles = []ScanOptions{
	{Name: "ocp4-cis-scan", Profile: "ocp4-cis"},
	{Name: "ocp4-moderate-scan", Profile: "ocp4-moderate"},
	{Name: "ocp4-pci-dss-scan", Profile: "ocp4-pci-dss"},
	{Name: "rhcos4-moderate-scan", Profile: "rhcos4-moderate"},
}

// CreateRecommendedScans creates scans for all recommended profiles.
func CreateRecommendedScans(ctx context.Context, client *k8s.Client, namespace string) ([]string, []error) {
	var created []string
	var errs []error

	for _, opts := range RecommendedProfiles {
		opts.Namespace = namespace
		if err := CreateScan(ctx, client, opts); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", opts.Profile, err))
		} else {
			created = append(created, opts.Name)
		}
	}

	return created, errs
}

// RescanSuite triggers a rescan of all ComplianceScans belonging to the named suite.
// It annotates each scan with compliance.openshift.io/rescan to trigger the operator.
func RescanSuite(ctx context.Context, client *k8s.Client, namespace, suiteName string) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}

	scans, err := client.Dynamic.Resource(complianceScanGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("compliance.openshift.io/suite=%s", suiteName),
		})
	if err != nil {
		return fmt.Errorf("listing ComplianceScans for suite %s: %w", suiteName, err)
	}

	if len(scans.Items) == 0 {
		return fmt.Errorf("no ComplianceScans found for suite %s", suiteName)
	}

	patch := []byte(`{"metadata":{"annotations":{"compliance.openshift.io/rescan":""}}}`)
	for _, scan := range scans.Items {
		_, err := client.Dynamic.Resource(complianceScanGVR).Namespace(namespace).
			Patch(ctx, scan.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("annotating ComplianceScan %s for rescan: %w", scan.GetName(), err)
		}
	}

	return nil
}

// DeleteScan deletes a ComplianceSuite and its matching ScanSettingBinding.
// Finalizers are removed first to prevent deletion from hanging.
func DeleteScan(ctx context.Context, client *k8s.Client, namespace, suiteName string) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}

	finalizerPatch := []byte(`{"metadata":{"finalizers":null}}`)

	// Remove finalizers and delete the ComplianceSuite
	_, err := client.Dynamic.Resource(complianceSuiteGVR).Namespace(namespace).
		Patch(ctx, suiteName, types.MergePatchType, finalizerPatch, metav1.PatchOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("removing finalizers from ComplianceSuite %s: %w", suiteName, err)
	}

	err = client.Dynamic.Resource(complianceSuiteGVR).Namespace(namespace).
		Delete(ctx, suiteName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("deleting ComplianceSuite %s: %w", suiteName, err)
	}

	// Remove finalizers and delete the matching ScanSettingBinding (non-fatal if not found or name mismatch)
	_, err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
		Patch(ctx, suiteName, types.MergePatchType, finalizerPatch, metav1.PatchOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Printf("Warning: could not remove finalizers from ScanSettingBinding %s: %v", suiteName, err)
	}

	err = client.Dynamic.Resource(scanSettingBindingGVR).Namespace(namespace).
		Delete(ctx, suiteName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Printf("Warning: could not delete ScanSettingBinding %s: %v", suiteName, err)
	}

	return nil
}

// ListProfiles returns all available compliance profiles.
func ListProfiles(ctx context.Context, client *k8s.Client, namespace string) ([]ProfileInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("kubernetes client is nil")
	}

	profiles, err := client.Dynamic.Resource(profileGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		if isCRDNotFound(err) {
			return []ProfileInfo{}, nil
		}
		return nil, fmt.Errorf("listing Profiles: %w", err)
	}

	var infos []ProfileInfo
	for _, p := range profiles.Items {
		title, _, _ := unstructured.NestedString(p.Object, "title")
		description, _, _ := unstructured.NestedString(p.Object, "description")
		infos = append(infos, ProfileInfo{
			Name:        p.GetName(),
			Title:       title,
			Description: description,
		})
	}

	return infos, nil
}
