package cmd

import (
	"os"
	"path/filepath"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/config"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "compliance-operator-dashboard",
	Short: "Web dashboard for OpenShift Compliance Operator",
	Long: `A web dashboard that provides a unified UI for managing the OpenShift
Compliance Operator. Supports operator installation, scan execution,
result visualization, and one-click remediation with real-time updates.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	defaultKubeconfig := os.Getenv("KUBECONFIG")
	if defaultKubeconfig == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultKubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	defaultNamespace := os.Getenv("COMPLIANCE_NAMESPACE")
	if defaultNamespace == "" {
		defaultNamespace = "openshift-compliance"
	}

	defaultCORef := os.Getenv("COMPLIANCE_OPERATOR_REF")

	defaultLogFormat := os.Getenv("LOG_FORMAT")
	if defaultLogFormat == "" {
		defaultLogFormat = "text"
	}

	rootCmd.PersistentFlags().StringVar(&cfg.KubeConfig, "kubeconfig", defaultKubeconfig,
		"Path to kubeconfig file (env: KUBECONFIG)")
	rootCmd.PersistentFlags().StringVar(&cfg.Namespace, "namespace", defaultNamespace,
		"Compliance Operator namespace (env: COMPLIANCE_NAMESPACE)")
	rootCmd.PersistentFlags().IntVar(&cfg.Port, "port", 8080,
		"HTTP server port")
	rootCmd.PersistentFlags().StringVar(&cfg.ComplianceOpRef, "co-ref", defaultCORef,
		"Compliance Operator version reference (env: COMPLIANCE_OPERATOR_REF, default: latest from GitHub)")
	rootCmd.PersistentFlags().StringVar(&cfg.LogFormat, "log-format", defaultLogFormat,
		"Log output format: text or json (env: LOG_FORMAT)")
}
