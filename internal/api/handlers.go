package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/compliance"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/ws"
)

// Handlers holds dependencies for API handlers.
type Handlers struct {
	k8sClient     *k8s.Client
	compliance    *compliance.Service
	hub           *ws.Hub
	namespace     string
	complianceRef string
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(client *k8s.Client, svc *compliance.Service, hub *ws.Hub, namespace, complianceRef string) *Handlers {
	return &Handlers{
		k8sClient:     client,
		compliance:    svc,
		hub:           hub,
		namespace:     namespace,
		complianceRef: complianceRef,
	}
}

// HandleClusterStatus returns cluster connectivity, version, and architecture info.
func (h *Handlers) HandleClusterStatus(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		writeJSON(w, http.StatusOK, compliance.ClusterStatus{Connected: false})
		return
	}

	status := compliance.ClusterStatus{
		Connected:     true,
		ServerVersion: h.k8sClient.ServerVersion,
	}

	if h.k8sClient.RestConfig != nil {
		status.ServerURL = h.k8sClient.RestConfig.Host
	}

	// Get node architecture info
	armNodes, _, _ := compliance.CheckARMCompatibility(r.Context(), h.k8sClient, h.complianceRef)
	status.ARMNodes = armNodes
	if armNodes > 0 {
		status.Architecture = "arm64"
	} else {
		status.Architecture = "amd64"
	}

	// Detect platform
	nodes, err := h.k8sClient.Clientset.CoreV1().Nodes().List(r.Context(), metav1.ListOptions{})
	if err == nil && len(nodes.Items) > 0 {
		for key := range nodes.Items[0].Labels {
			if strings.Contains(key, "openshift") {
				status.Platform = "OpenShift"
				break
			}
		}
		if status.Platform == "" {
			status.Platform = "Kubernetes"
		}
	}

	writeJSON(w, http.StatusOK, status)
}

// HandleOperatorInstall starts the operator installation process.
func (h *Handlers) HandleOperatorInstall(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Not connected to Kubernetes cluster")
		return
	}

	progress := make(chan compliance.InstallProgress, 32)

	// Use a background context — the request context will be canceled
	// as soon as the 202 response is sent, but install is long-running.
	installCtx := context.Background()

	go func() {
		compliance.Install(installCtx, h.k8sClient, h.namespace, h.complianceRef, progress)
	}()

	// Stream progress to WebSocket
	go func() {
		for p := range progress {
			h.hub.Broadcast(ws.Message{
				Type:    ws.MessageTypeInstallProgress,
				Payload: p,
			})
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Installation started. Follow progress via WebSocket.",
	})
}

// HandleOperatorStatus returns the current operator status.
func (h *Handlers) HandleOperatorStatus(w http.ResponseWriter, r *http.Request) {
	status, err := compliance.GetStatus(r.Context(), h.k8sClient, h.namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// HandleCreateScan creates a new compliance scan.
func (h *Handlers) HandleCreateScan(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Not connected to Kubernetes cluster")
		return
	}

	var opts compliance.ScanOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if opts.Name == "" {
		opts.Name = "cis-scan"
	}
	if opts.Profile == "" {
		opts.Profile = "ocp4-cis"
	}
	if opts.Namespace == "" {
		opts.Namespace = h.namespace
	}

	if err := compliance.CreateScan(r.Context(), h.k8sClient, opts); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "Scan created successfully",
		"name":    opts.Name,
	})
}

// HandleListScans returns the status of all scans.
func (h *Handlers) HandleListScans(w http.ResponseWriter, r *http.Request) {
	statuses, err := compliance.GetScanStatus(r.Context(), h.k8sClient, h.namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, statuses)
}

// HandleListProfiles returns all available compliance profiles.
func (h *Handlers) HandleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := compliance.ListProfiles(r.Context(), h.k8sClient, h.namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

// HandleCreateRecommendedScans creates scans for the 4 recommended profiles.
func (h *Handlers) HandleCreateRecommendedScans(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Not connected to Kubernetes cluster")
		return
	}

	created, errs := compliance.CreateRecommendedScans(r.Context(), h.k8sClient, h.namespace)

	var errMsgs []string
	for _, e := range errs {
		errMsgs = append(errMsgs, e.Error())
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": fmt.Sprintf("Created %d recommended scans", len(created)),
		"created": created,
		"errors":  errMsgs,
	})
}

// HandleGetResults returns full compliance results with optional filtering.
func (h *Handlers) HandleGetResults(w http.ResponseWriter, r *http.Request) {
	severity := r.URL.Query().Get("severity")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	if severity == "" && status == "" && search == "" {
		// Return full results
		data, err := compliance.GetComplianceResults(r.Context(), h.k8sClient, h.namespace)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, data)
		return
	}

	// Filtered results
	results, err := compliance.GetFilteredResults(r.Context(), h.k8sClient, h.namespace, severity, status, search)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// HandleGetCheckResult returns detail for a single check result.
func (h *Handlers) HandleGetCheckResult(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Check name is required")
		return
	}

	detail, err := compliance.GetCheckResult(r.Context(), h.k8sClient, h.namespace, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// HandleGetResultsSummary returns only the summary counts.
func (h *Handlers) HandleGetResultsSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := compliance.GetResultsSummary(r.Context(), h.k8sClient, h.namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// HandleApplyRemediation applies a single remediation.
func (h *Handlers) HandleApplyRemediation(w http.ResponseWriter, r *http.Request) {
	if h.k8sClient == nil {
		writeError(w, http.StatusServiceUnavailable, "Not connected to Kubernetes cluster")
		return
	}

	// Extract name from path: /api/remediate/{name}
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Remediation name is required")
		return
	}

	result, err := compliance.ApplyRemediation(r.Context(), h.k8sClient, h.namespace, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast result via WebSocket
	h.hub.Broadcast(ws.Message{
		Type:    ws.MessageTypeRemediationResult,
		Payload: result,
	})

	writeJSON(w, http.StatusOK, result)
}

// HandleGetRemediation returns detail for a single remediation including its YAML.
func (h *Handlers) HandleGetRemediation(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Remediation name is required")
		return
	}

	detail, err := compliance.GetRemediation(r.Context(), h.k8sClient, h.namespace, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

// HandleListRemediations lists all available remediations.
func (h *Handlers) HandleListRemediations(w http.ResponseWriter, r *http.Request) {
	remediations, err := compliance.ListRemediations(r.Context(), h.k8sClient, h.namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, remediations)
}

// HandleWebSocket upgrades to WebSocket connection.
func (h *Handlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws.ServeWS(h.hub, w, r)
}
