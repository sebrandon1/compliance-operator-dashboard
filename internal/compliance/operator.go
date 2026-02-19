package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

const (
	coRepoOwner      = "ComplianceAsCode"
	coRepoName       = "compliance-operator"
	operatorName     = "compliance-operator"
	subscriptionName = "compliance-operator-sub"

	marketplaceNS = "openshift-marketplace"
)

var (
	subscriptionGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "subscriptions",
	}
	csvGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "clusterserviceversions",
	}
	catalogSourceGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1alpha1", Resource: "catalogsources",
	}
	operatorGroupGVR = schema.GroupVersionResource{
		Group: "operators.coreos.com", Version: "v1", Resource: "operatorgroups",
	}
	packageManifestGVR = schema.GroupVersionResource{
		Group: "packages.operators.coreos.com", Version: "v1", Resource: "packagemanifests",
	}
	profileBundleGVR = schema.GroupVersionResource{
		Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "profilebundles",
	}
)

// GetLatestRelease fetches the latest Compliance Operator release tag from GitHub.
func GetLatestRelease(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", coRepoOwner, coRepoName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	return release.TagName, nil
}

// CheckMarketplaceHealth verifies that pods in openshift-marketplace are healthy.
func CheckMarketplaceHealth(ctx context.Context, client *k8s.Client) error {
	if client == nil {
		return fmt.Errorf("kubernetes client is nil")
	}

	// Check namespace exists
	_, err := client.Clientset.CoreV1().Namespaces().Get(ctx, marketplaceNS, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("namespace %s not found: %w", marketplaceNS, err)
	}

	// Check for pods in error states
	pods, err := client.Clientset.CoreV1().Pods(marketplaceNS).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods in %s: %w", marketplaceNS, err)
	}

	errorStates := []string{"ImagePullBackOff", "ErrImagePull", "CrashLoopBackOff",
		"CreateContainerConfigError", "InvalidImageName"}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning {
			continue
		}
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				for _, errState := range errorStates {
					if cs.State.Waiting.Reason == errState {
						return fmt.Errorf("pod %s in error state: %s", pod.Name, cs.State.Waiting.Reason)
					}
				}
			}
		}
	}

	return nil
}

// CheckRedHatOperator checks if the Red Hat certified operator is available.
func CheckRedHatOperator(ctx context.Context, client *k8s.Client) (bool, error) {
	if client == nil {
		return false, fmt.Errorf("kubernetes client is nil")
	}

	// Check if redhat-operators catalog source exists
	_, err := client.Dynamic.Resource(catalogSourceGVR).Namespace(marketplaceNS).
		Get(ctx, "redhat-operators", metav1.GetOptions{})
	if err != nil {
		return false, nil // Not an error, just not available
	}

	// Check if compliance-operator package is available from redhat-operators
	pm, err := client.Dynamic.Resource(packageManifestGVR).Namespace(marketplaceNS).
		Get(ctx, "compliance-operator", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}

	catalogSource, _, _ := unstructured.NestedString(pm.Object, "status", "catalogSource")
	return catalogSource == "redhat-operators", nil
}

// CheckARMCompatibility verifies ARM64 compatibility for the given version.
func CheckARMCompatibility(ctx context.Context, client *k8s.Client, coRef string) (int, bool, error) {
	if client == nil {
		return 0, false, fmt.Errorf("kubernetes client is nil")
	}

	nodes, err := client.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, false, fmt.Errorf("listing nodes: %w", err)
	}

	armNodes := 0
	for _, node := range nodes.Items {
		if node.Status.NodeInfo.Architecture == "arm64" {
			armNodes++
		}
	}

	if armNodes == 0 {
		return 0, true, nil
	}

	// Versions before v1.7.0 don't support ARM64
	if strings.HasPrefix(coRef, "v1.") {
		parts := strings.SplitN(strings.TrimPrefix(coRef, "v1."), ".", 2)
		if len(parts) > 0 {
			minor := parts[0]
			if minor >= "0" && minor < "7" {
				return armNodes, false, nil
			}
		}
	}

	return armNodes, true, nil
}

// Install performs the full Compliance Operator installation.
// It sends progress updates to the provided channel.
func Install(ctx context.Context, client *k8s.Client, namespace, coRef string, progress chan<- InstallProgress) {
	defer close(progress)

	sendProgress := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message}
	}
	sendError := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message, Error: message, Done: true}
	}
	sendDone := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message, Done: true}
	}

	if client == nil {
		sendError("init", "Kubernetes client is not connected")
		return
	}

	// Step 1: Check marketplace health
	sendProgress("marketplace", "Checking marketplace health...")
	if err := CheckMarketplaceHealth(ctx, client); err != nil {
		sendError("marketplace", fmt.Sprintf("Marketplace health check failed: %v", err))
		return
	}
	sendProgress("marketplace", "Marketplace is healthy")

	// Step 2: Resolve version
	if coRef == "" {
		sendProgress("version", "Resolving latest release from GitHub...")
		var err error
		coRef, err = GetLatestRelease(ctx)
		if err != nil {
			log.Printf("Could not fetch latest release: %v, falling back to master", err)
			coRef = "master"
		}
	}
	sendProgress("version", fmt.Sprintf("Using Compliance Operator ref: %s", coRef))

	// Step 3: Check ARM compatibility
	sendProgress("arch", "Checking cluster architecture...")
	armNodes, compatible, err := CheckARMCompatibility(ctx, client, coRef)
	if err != nil {
		sendError("arch", fmt.Sprintf("Architecture check failed: %v", err))
		return
	}
	if !compatible {
		sendError("arch", fmt.Sprintf("Version %s does not support ARM64 (%d ARM nodes detected). Use v1.7.0+.", coRef, armNodes))
		return
	}
	if armNodes > 0 {
		sendProgress("arch", fmt.Sprintf("ARM64 compatible (%d ARM nodes)", armNodes))
	} else {
		sendProgress("arch", "x86_64 cluster detected")
	}

	// Step 4: Create namespace
	sendProgress("namespace", fmt.Sprintf("Creating namespace %s...", namespace))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}
	_, err = client.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		sendError("namespace", fmt.Sprintf("Failed to create namespace: %v", err))
		return
	}
	sendProgress("namespace", fmt.Sprintf("Namespace %s ready", namespace))

	// Step 5: Detect install source
	sendProgress("source", "Checking for Red Hat certified operator...")
	useRedHat, err := CheckRedHatOperator(ctx, client)
	if err != nil {
		log.Printf("Red Hat operator check error: %v, falling back to community", err)
	}

	// Step 6: Install operator
	if useRedHat {
		sendProgress("install", "Installing Red Hat certified Compliance Operator...")
		if err := installRedHatOperator(ctx, client, namespace); err != nil {
			sendError("install", fmt.Sprintf("Red Hat operator install failed: %v", err))
			return
		}
	} else {
		sendProgress("install", "Installing community Compliance Operator...")
		if err := installCommunityOperator(ctx, client, namespace, coRef); err != nil {
			sendError("install", fmt.Sprintf("Community operator install failed: %v", err))
			return
		}
	}

	// Step 7: Wait for CSV
	sendProgress("csv", "Waiting for ClusterServiceVersion...")
	csvName, err := waitForCSV(ctx, client, namespace)
	if err != nil {
		sendError("csv", fmt.Sprintf("CSV wait failed: %v", err))
		return
	}
	sendProgress("csv", fmt.Sprintf("CSV %s succeeded", csvName))

	// Step 8: Apply supplemental RBAC
	sendProgress("rbac", "Applying supplemental RBAC for Job creation...")
	if err := applySupplementalRBAC(ctx, client, namespace); err != nil {
		log.Printf("Warning: supplemental RBAC failed: %v", err)
	}
	sendProgress("rbac", "Supplemental RBAC applied")

	// Step 9: Wait for pods
	sendProgress("pods", "Waiting for operator pods to be ready...")
	if err := waitForPodsReady(ctx, client, namespace); err != nil {
		log.Printf("Warning: some pods may not be ready: %v", err)
	}
	sendProgress("pods", "Operator pods are ready")

	// Step 10: Wait for ProfileBundles
	sendProgress("bundles", "Waiting for ProfileBundles to become VALID...")
	if err := waitForProfileBundles(ctx, client, namespace); err != nil {
		log.Printf("Warning: ProfileBundles may not be valid: %v", err)
	}

	sendDone("complete", "Compliance Operator installed successfully")
}

// Uninstall removes the Compliance Operator and all its resources.
// It sends progress updates to the provided channel.
func Uninstall(ctx context.Context, client *k8s.Client, namespace string, progress chan<- InstallProgress) {
	defer close(progress)

	sendProgress := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message}
	}
	sendError := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message, Error: message, Done: true}
	}
	sendDone := func(step, message string) {
		progress <- InstallProgress{Step: step, Message: message, Done: true}
	}

	if client == nil {
		sendError("init", "Kubernetes client is not connected")
		return
	}

	finalizerPatch := []byte(`{"metadata":{"finalizers":null}}`)

	// Step 1: Delete all Compliance CRs (remove finalizers first)
	complianceCRDs := []struct {
		name string
		gvr  schema.GroupVersionResource
	}{
		{"ComplianceCheckResults", complianceCheckResultGVR},
		{"ComplianceRemediations", complianceRemediationGVR},
		{"ComplianceSuites", complianceSuiteGVR},
		{"ComplianceScans", complianceScanGVR},
		{"ScanSettingBindings", scanSettingBindingGVR},
		{"ScanSettings", scanSettingGVR},
		{"ProfileBundles", profileBundleGVR},
		{"Profiles", profileGVR},
	}

	for _, crd := range complianceCRDs {
		sendProgress("cleanup", fmt.Sprintf("Removing %s...", crd.name))
		items, err := client.Dynamic.Resource(crd.gvr).Namespace(namespace).
			List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Warning: listing %s: %v", crd.name, err)
			continue
		}
		for _, item := range items.Items {
			// Remove finalizers
			_, _ = client.Dynamic.Resource(crd.gvr).Namespace(namespace).
				Patch(ctx, item.GetName(), types.MergePatchType, finalizerPatch, metav1.PatchOptions{})
			// Delete
			_ = client.Dynamic.Resource(crd.gvr).Namespace(namespace).
				Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		}
	}
	sendProgress("cleanup", "Compliance resources removed")

	// Step 2: Delete Subscription
	sendProgress("subscription", "Deleting Subscription...")
	err := client.Dynamic.Resource(subscriptionGVR).Namespace(namespace).
		Delete(ctx, subscriptionName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Printf("Warning: deleting Subscription: %v", err)
	}
	sendProgress("subscription", "Subscription deleted")

	// Step 3: Delete CSV
	sendProgress("csv", "Deleting ClusterServiceVersion...")
	csvs, err := client.Dynamic.Resource(csvGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, csv := range csvs.Items {
			_ = client.Dynamic.Resource(csvGVR).Namespace(namespace).
				Delete(ctx, csv.GetName(), metav1.DeleteOptions{})
		}
	}
	sendProgress("csv", "ClusterServiceVersion deleted")

	// Step 4: Delete OperatorGroup
	sendProgress("operatorgroup", "Deleting OperatorGroup...")
	err = client.Dynamic.Resource(operatorGroupGVR).Namespace(namespace).
		Delete(ctx, operatorName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Printf("Warning: deleting OperatorGroup: %v", err)
	}
	sendProgress("operatorgroup", "OperatorGroup deleted")

	// Step 5: Delete CatalogSource (community install)
	sendProgress("catalogsource", "Deleting CatalogSource...")
	err = client.Dynamic.Resource(catalogSourceGVR).Namespace(marketplaceNS).
		Delete(ctx, operatorName, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		log.Printf("Warning: deleting CatalogSource: %v", err)
	}
	sendProgress("catalogsource", "CatalogSource deleted")

	// Step 6: Delete namespace
	sendProgress("namespace", fmt.Sprintf("Deleting namespace %s...", namespace))
	err = client.Clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		sendError("namespace", fmt.Sprintf("Failed to delete namespace: %v", err))
		return
	}

	// Wait for namespace deletion
	for i := 0; i < 30; i++ {
		_, err := client.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			break // Namespace is gone
		}
		select {
		case <-ctx.Done():
			sendError("namespace", "Timed out waiting for namespace deletion")
			return
		case <-time.After(5 * time.Second):
		}
	}
	sendProgress("namespace", "Namespace deleted")

	sendDone("complete", "Compliance Operator uninstalled successfully")
}

// GetStatus returns the current status of the Compliance Operator.
func GetStatus(ctx context.Context, client *k8s.Client, namespace string) (*OperatorStatus, error) {
	if client == nil {
		return &OperatorStatus{Installed: false}, nil
	}

	status := &OperatorStatus{}

	// Check for subscription
	sub, err := client.Dynamic.Resource(subscriptionGVR).Namespace(namespace).
		Get(ctx, subscriptionName, metav1.GetOptions{})
	if err != nil {
		return status, nil // Not installed
	}

	csvName, _, _ := unstructured.NestedString(sub.Object, "status", "installedCSV")
	if csvName == "" {
		return status, nil
	}

	status.Installed = true
	status.Version = csvName

	// Check CSV phase
	csv, err := client.Dynamic.Resource(csvGVR).Namespace(namespace).
		Get(ctx, csvName, metav1.GetOptions{})
	if err == nil {
		phase, _, _ := unstructured.NestedString(csv.Object, "status", "phase")
		status.CSVPhase = phase
	}

	// Get pod statuses
	pods, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			ps := PodStatus{
				Name:  pod.Name,
				Phase: string(pod.Status.Phase),
			}
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ps.Ready = true
					break
				}
			}
			status.Pods = append(status.Pods, ps)
		}
	}

	// Get ProfileBundle statuses
	bundles, err := client.Dynamic.Resource(profileBundleGVR).Namespace(namespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, bundle := range bundles.Items {
			dsStatus, _, _ := unstructured.NestedString(bundle.Object, "status", "dataStreamStatus")
			status.ProfileBundles = append(status.ProfileBundles, BundleStatus{
				Name:             bundle.GetName(),
				DataStreamStatus: dsStatus,
			})
		}
	}

	return status, nil
}

func installRedHatOperator(ctx context.Context, client *k8s.Client, namespace string) error {
	// Create OperatorGroup
	og := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1",
			"kind":       "OperatorGroup",
			"metadata": map[string]interface{}{
				"name":      operatorName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"targetNamespaces": []interface{}{namespace},
			},
		},
	}
	_, err := client.Dynamic.Resource(operatorGroupGVR).Namespace(namespace).
		Create(ctx, og, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating OperatorGroup: %w", err)
	}

	// Create Subscription
	sub := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]interface{}{
				"name":      subscriptionName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"channel":             "stable",
				"installPlanApproval": "Automatic",
				"name":               operatorName,
				"source":             "redhat-operators",
				"sourceNamespace":    marketplaceNS,
			},
		},
	}
	_, err = client.Dynamic.Resource(subscriptionGVR).Namespace(namespace).
		Create(ctx, sub, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating Subscription: %w", err)
	}

	return nil
}

func installCommunityOperator(ctx context.Context, client *k8s.Client, namespace, coRef string) error {
	// Create CatalogSource
	cs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "CatalogSource",
			"metadata": map[string]interface{}{
				"name":      operatorName,
				"namespace": marketplaceNS,
			},
			"spec": map[string]interface{}{
				"displayName": "Compliance Operator Upstream",
				"image":       fmt.Sprintf("ghcr.io/complianceascode/compliance-operator-catalog:%s", coRef),
				"publisher":   "github.com/complianceascode/compliance-operator",
				"sourceType":  "grpc",
				"grpcPodConfig": map[string]interface{}{
					"tolerations": []interface{}{
						map[string]interface{}{
							"key":      "node-role.kubernetes.io/master",
							"operator": "Exists",
							"effect":   "NoSchedule",
						},
						map[string]interface{}{
							"key":      "node-role.kubernetes.io/control-plane",
							"operator": "Exists",
							"effect":   "NoSchedule",
						},
					},
				},
			},
		},
	}
	_, err := client.Dynamic.Resource(catalogSourceGVR).Namespace(marketplaceNS).
		Create(ctx, cs, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating CatalogSource: %w", err)
	}

	// Create OperatorGroup
	og := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1",
			"kind":       "OperatorGroup",
			"metadata": map[string]interface{}{
				"name":      operatorName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"targetNamespaces": []interface{}{namespace},
			},
		},
	}
	_, err = client.Dynamic.Resource(operatorGroupGVR).Namespace(namespace).
		Create(ctx, og, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating OperatorGroup: %w", err)
	}

	// Create Subscription
	sub := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "operators.coreos.com/v1alpha1",
			"kind":       "Subscription",
			"metadata": map[string]interface{}{
				"name":      subscriptionName,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"channel":             "alpha",
				"installPlanApproval": "Automatic",
				"name":               operatorName,
				"source":             operatorName,
				"sourceNamespace":    marketplaceNS,
			},
		},
	}
	_, err = client.Dynamic.Resource(subscriptionGVR).Namespace(namespace).
		Create(ctx, sub, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating Subscription: %w", err)
	}

	return nil
}

func waitForCSV(ctx context.Context, client *k8s.Client, namespace string) (string, error) {
	var csvName string

	// Wait for subscription to populate installedCSV
	for i := 0; i < 30; i++ {
		sub, err := client.Dynamic.Resource(subscriptionGVR).Namespace(namespace).
			Get(ctx, subscriptionName, metav1.GetOptions{})
		if err == nil {
			name, _, _ := unstructured.NestedString(sub.Object, "status", "installedCSV")
			if name != "" {
				csvName = name
				break
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}

	if csvName == "" {
		return "", fmt.Errorf("installedCSV not populated after timeout")
	}

	// Wait for CSV to reach Succeeded phase
	for i := 0; i < 30; i++ {
		csv, err := client.Dynamic.Resource(csvGVR).Namespace(namespace).
			Get(ctx, csvName, metav1.GetOptions{})
		if err == nil {
			phase, _, _ := unstructured.NestedString(csv.Object, "status", "phase")
			if phase == "Succeeded" {
				return csvName, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}

	return csvName, fmt.Errorf("CSV %s did not reach Succeeded phase", csvName)
}

func applySupplementalRBAC(ctx context.Context, client *k8s.Client, namespace string) error {
	// Create Role for Job permissions
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compliance-operator-job-permissions",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "update", "patch"},
			},
		},
	}
	_, err := client.Clientset.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating Role: %w", err)
	}

	// Create RoleBinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compliance-operator-job-permissions",
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "compliance-operator-job-permissions",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      operatorName,
				Namespace: namespace,
			},
		},
	}
	_, err = client.Clientset.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("creating RoleBinding: %w", err)
	}

	return nil
}

func waitForPodsReady(ctx context.Context, client *k8s.Client, namespace string) error {
	for i := 0; i < 30; i++ {
		pods, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("listing pods: %w", err)
		}

		allReady := true
		hasPods := false
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			hasPods = true
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			if !ready {
				allReady = false
			}
		}

		if hasPods && allReady {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}

	return fmt.Errorf("pods not ready after timeout")
}

func waitForProfileBundles(ctx context.Context, client *k8s.Client, namespace string) error {
	for i := 0; i < 30; i++ {
		bundles, err := client.Dynamic.Resource(profileBundleGVR).Namespace(namespace).
			List(ctx, metav1.ListOptions{})
		if err != nil || len(bundles.Items) == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
			}
			continue
		}

		allValid := true
		for _, bundle := range bundles.Items {
			dsStatus, _, _ := unstructured.NestedString(bundle.Object, "status", "dataStreamStatus")
			if dsStatus != "VALID" {
				allValid = false
				break
			}
		}

		if allValid {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}

	return fmt.Errorf("ProfileBundles not VALID after timeout")
}
