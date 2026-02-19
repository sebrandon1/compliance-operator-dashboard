package api

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/compliance"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/config"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
	"github.com/sebrandon1/compliance-operator-dashboard/internal/ws"
)

//go:embed all:frontend_dist
var frontendFS embed.FS

// Server is the HTTP server for the dashboard.
type Server struct {
	handlers *Handlers
	hub      *ws.Hub
}

// NewServer creates a new Server instance.
func NewServer(cfg config.Config, svc *compliance.Service, hub *ws.Hub) *Server {
	// Extract k8s client from service - may be nil if not connected
	var k8sClient *k8s.Client
	if svc != nil {
		k8sClient = svc.K8sClient()
	}

	return &Server{
		handlers: NewHandlers(k8sClient, svc, hub, cfg.Namespace, cfg.ComplianceOpRef),
		hub:      hub,
	}
}

// Handler returns the configured HTTP handler with all routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes (Go 1.22+ method routing)
	mux.HandleFunc("GET /api/cluster/status", s.handlers.HandleClusterStatus)
	mux.HandleFunc("POST /api/operator/install", s.handlers.HandleOperatorInstall)
	mux.HandleFunc("GET /api/operator/status", s.handlers.HandleOperatorStatus)
	mux.HandleFunc("DELETE /api/operator", s.handlers.HandleUninstallOperator)
	mux.HandleFunc("POST /api/scans/recommended", s.handlers.HandleCreateRecommendedScans)
	mux.HandleFunc("POST /api/scans/{name}/rescan", s.handlers.HandleRescan)
	mux.HandleFunc("DELETE /api/scans/{name}", s.handlers.HandleDeleteScan)
	mux.HandleFunc("POST /api/scans", s.handlers.HandleCreateScan)
	mux.HandleFunc("GET /api/scans", s.handlers.HandleListScans)
	mux.HandleFunc("GET /api/profiles", s.handlers.HandleListProfiles)
	mux.HandleFunc("GET /api/results/summary", s.handlers.HandleGetResultsSummary)
	mux.HandleFunc("GET /api/results/{name}", s.handlers.HandleGetCheckResult)
	mux.HandleFunc("GET /api/results", s.handlers.HandleGetResults)
	mux.HandleFunc("POST /api/remediate/{name}", s.handlers.HandleApplyRemediation)
	mux.HandleFunc("GET /api/remediations/{name}", s.handlers.HandleGetRemediation)
	mux.HandleFunc("GET /api/remediations", s.handlers.HandleListRemediations)
	mux.HandleFunc("GET /ws/watch", s.handlers.HandleWebSocket)

	// Serve embedded frontend (SPA fallback)
	mux.Handle("/", spaHandler())

	// Apply middleware
	handler := recoveryMiddleware(loggingMiddleware(corsMiddleware(mux)))
	return handler
}

// spaHandler serves the embedded React SPA with fallback to index.html.
func spaHandler() http.Handler {
	distFS, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		log.Printf("Warning: embedded frontend not available: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html><html><body>
				<h1>Compliance Operator Dashboard</h1>
				<p>Frontend not built. Run <code>make frontend-build</code> first.</p>
			</body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to serve the file directly
		if path != "/" && !strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/ws") {
			// Check if file exists
			f, err := distFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all unknown paths
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
