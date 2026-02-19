package compliance

import (
	"time"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

// Severity represents a compliance check severity level.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// CheckStatus represents the status of a compliance check result.
type CheckStatus string

const (
	CheckStatusPass          CheckStatus = "PASS"
	CheckStatusFail          CheckStatus = "FAIL"
	CheckStatusManual        CheckStatus = "MANUAL"
	CheckStatusSkip          CheckStatus = "SKIP"
	CheckStatusNotApplicable CheckStatus = "NOT-APPLICABLE"
)

// CheckResult represents a single compliance check result.
type CheckResult struct {
	Name        string      `json:"name"`
	Check       string      `json:"check"`
	Status      CheckStatus `json:"status"`
	Description string      `json:"description"`
	Severity    Severity    `json:"severity"`
	ScanName    string      `json:"scan_name,omitempty"`
	Suite       string      `json:"suite,omitempty"`
}

// CheckResultDetail is a full check result with instructions, rationale, and remediation link.
type CheckResultDetail struct {
	CheckResult
	ID              string `json:"id"`
	Instructions    string `json:"instructions"`
	Rationale       string `json:"rationale"`
	HasRemediation  bool   `json:"has_remediation"`
	RemediationName string `json:"remediation_name,omitempty"`
}

// SeverityGroup holds check results for a single severity level.
type SeverityGroup struct {
	Severity Severity      `json:"severity"`
	Count    int           `json:"count"`
	Checks   []CheckResult `json:"checks"`
}

// Summary holds aggregate counts for compliance results.
type Summary struct {
	TotalChecks int `json:"total_checks"`
	Passing     int `json:"passing"`
	Failing     int `json:"failing"`
	Manual      int `json:"manual"`
	Skipped     int `json:"skipped"`
}

// ComplianceData is the top-level compliance results structure.
type ComplianceData struct {
	ScanDate      string        `json:"scan_date"`
	Summary       Summary       `json:"summary"`
	Remediations  SeverityMap   `json:"remediations"`
	PassingChecks SeverityMap   `json:"passing_checks"`
	ManualChecks  []CheckResult `json:"manual_checks"`
}

// SeverityMap groups check results by severity.
type SeverityMap struct {
	High   []CheckResult `json:"high"`
	Medium []CheckResult `json:"medium"`
	Low    []CheckResult `json:"low"`
}

// OperatorStatus represents the current state of the Compliance Operator.
type OperatorStatus struct {
	Installed      bool           `json:"installed"`
	Version        string         `json:"version,omitempty"`
	CSVPhase       string         `json:"csv_phase,omitempty"`
	Pods           []PodStatus    `json:"pods,omitempty"`
	ProfileBundles []BundleStatus `json:"profile_bundles,omitempty"`
}

// PodStatus represents a pod's status summary.
type PodStatus struct {
	Name   string `json:"name"`
	Phase  string `json:"phase"`
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// BundleStatus represents a ProfileBundle's status.
type BundleStatus struct {
	Name             string `json:"name"`
	DataStreamStatus string `json:"data_stream_status"`
}

// InstallProgress represents a step in the operator installation process.
type InstallProgress struct {
	Step    string `json:"step"`
	Message string `json:"message"`
	Done    bool   `json:"done"`
	Error   string `json:"error,omitempty"`
}

// InstallSource indicates whether using Red Hat certified or community operator.
type InstallSource string

const (
	InstallSourceRedHat    InstallSource = "redhat"
	InstallSourceCommunity InstallSource = "community"
)

// ProfileInfo represents an available compliance profile.
type ProfileInfo struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ScanOptions configures a one-off compliance scan.
type ScanOptions struct {
	Name      string `json:"name"`
	Profile   string `json:"profile"`
	Namespace string `json:"namespace,omitempty"`
}

// PeriodicScanOptions configures a periodic compliance scan.
type PeriodicScanOptions struct {
	Schedule         string   `json:"schedule"`
	Profiles         []string `json:"profiles"`
	Namespace        string   `json:"namespace,omitempty"`
	StorageClassName string   `json:"storage_class_name,omitempty"`
	StorageSize      string   `json:"storage_size,omitempty"`
	Rotation         int      `json:"rotation,omitempty"`
	Roles            []string `json:"roles,omitempty"`
}

// ScanStatus represents the status of a compliance scan.
type ScanStatus struct {
	Name           string `json:"name"`
	Phase          string `json:"phase"`
	Result         string `json:"result,omitempty"`
	Profile        string `json:"profile,omitempty"`
	ScanType       string `json:"scan_type,omitempty"`
	ContentImage   string `json:"content_image,omitempty"`
	StartTimestamp string `json:"start_timestamp,omitempty"`
	EndTimestamp   string `json:"end_timestamp,omitempty"`
	Warnings       string `json:"warnings,omitempty"`
}

// SuiteStatus represents the status of a compliance suite.
type SuiteStatus struct {
	Name        string       `json:"name"`
	Phase       string       `json:"phase"`
	Scans       []ScanStatus `json:"scans,omitempty"`
	Result      string       `json:"result,omitempty"`
	CreatedAt   string       `json:"created_at,omitempty"`
	Conditions  []Condition  `json:"conditions,omitempty"`
}

// Condition represents a K8s-style status condition.
type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	LastTransitionTime string `json:"last_transition_time,omitempty"`
}

// RemediationInfo represents a single compliance remediation.
type RemediationInfo struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Severity     Severity `json:"severity"`
	Applied      bool     `json:"applied"`
	RebootNeeded bool     `json:"reboot_needed"`
	Role         string   `json:"role,omitempty"`
}

// RemediationDetail is a full remediation with its object YAML.
type RemediationDetail struct {
	RemediationInfo
	ObjectYAML string `json:"object_yaml"`
	APIVersion string `json:"api_version,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

// RemediationResult is the outcome of applying a remediation.
type RemediationResult struct {
	Name    string `json:"name"`
	Applied bool   `json:"applied"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// StorageInfo represents detected storage information.
type StorageInfo struct {
	HasDefaultStorageClass bool   `json:"has_default_storage_class"`
	StorageClassName       string `json:"storage_class_name,omitempty"`
	Provisioner            string `json:"provisioner,omitempty"`
	HostpathCSIDeployed    bool   `json:"hostpath_csi_deployed"`
	Recommendation         string `json:"recommendation,omitempty"`
}

// ClusterStatus represents the current cluster connection status.
type ClusterStatus struct {
	Connected     bool   `json:"connected"`
	ServerURL     string `json:"server_url,omitempty"`
	ServerVersion string `json:"server_version,omitempty"`
	Platform      string `json:"platform,omitempty"`
	Architecture  string `json:"architecture,omitempty"`
	ARMNodes      int    `json:"arm_nodes"`
}

// Service provides compliance operator operations.
type Service struct {
	k8sClient     *k8s.Client
	namespace     string
	complianceRef string
}

// NewService creates a new compliance Service.
func NewService(k8sClient *k8s.Client, namespace, complianceRef string) *Service {
	return &Service{
		k8sClient:     k8sClient,
		namespace:     namespace,
		complianceRef: complianceRef,
	}
}

// K8sClient returns the underlying Kubernetes client.
func (s *Service) K8sClient() *k8s.Client {
	if s == nil {
		return nil
	}
	return s.k8sClient
}

// DefaultPeriodicScanOptions returns sensible defaults for periodic scans.
func DefaultPeriodicScanOptions(namespace string) PeriodicScanOptions {
	return PeriodicScanOptions{
		Schedule:    "0 1 * * *",
		Profiles:    []string{"ocp4-cis", "ocp4-e8", "rhcos4-e8"},
		Namespace:   namespace,
		StorageSize: "1Gi",
		Rotation:    3,
		Roles:       []string{"worker", "master"},
	}
}

// ScanTimestamp returns a formatted scan timestamp.
func ScanTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
