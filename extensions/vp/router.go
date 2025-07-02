package vp

import (
	"net/http"

	"github.com/modelcontextprotocol/registry/extensions/vp/handlers"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// RegisterRoutes registers all /vp API routes to the provided router
func RegisterRoutes(mux *http.ServeMux, registry service.RegistryService) {
	// Register vp endpoints with filtering support
	mux.HandleFunc("/vp/servers", handlers.ServersHandler(registry))
	mux.HandleFunc("/vp/servers/{id}", handlers.ServersDetailHandler(registry))
	
	// Test endpoint
	mux.HandleFunc("/vp/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","message":"VP endpoints are working"}`))
	})
}