package config

// Config holds the application configuration.
type Config struct {
	KubeConfig      string
	Namespace       string
	Port            int
	ComplianceOpRef string
}
