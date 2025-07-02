package extensions

import (
	"net/http"

	"github.com/modelcontextprotocol/registry/extensions/vp"
	"github.com/modelcontextprotocol/registry/internal/api/router"
	"github.com/modelcontextprotocol/registry/internal/auth"
	"github.com/modelcontextprotocol/registry/internal/config"
	"github.com/modelcontextprotocol/registry/internal/middleware"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// NewWithExtensions creates a new router with all API versions including extensions
func NewWithExtensions(cfg *config.Config, registry service.RegistryService, authService auth.Service) http.Handler {
	// Get the base router with v0 routes
	mux := router.New(cfg, registry, authService)
	
	// Register vp extension routes
	vp.RegisterRoutes(mux, registry)
	
	// Apply CORS middleware
	handler := middleware.CORS(cfg.CORSOrigins)(mux)
	
	return handler
}