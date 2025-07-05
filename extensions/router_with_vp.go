package extensions

import (
	"net/http"
	"time"

	"github.com/modelcontextprotocol/registry/extensions/vp"
	"github.com/modelcontextprotocol/registry/internal/auth"
	"github.com/modelcontextprotocol/registry/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
)

// ExtendedRouterConfig holds configuration for extended router
type ExtendedRouterConfig struct {
	BaseRouter       *http.ServeMux
	Service          service.RegistryService
	MongoClient      *mongo.Client
	DatabaseName     string
	AnalyticsBaseURL string
	AuthService      auth.Service
}

// SetupExtendedRouter adds VP routes to the existing router
func SetupExtendedRouter(config ExtendedRouterConfig) error {
	// Configure VP routes
	vpConfig := vp.Config{
		Service:          config.Service,
		MongoClient:      config.MongoClient,
		DatabaseName:     config.DatabaseName,
		CacheTTL:         5 * time.Minute,
		AnalyticsBaseURL: config.AnalyticsBaseURL,
		AuthService:      config.AuthService,
	}

	// Setup VP routes on the existing router
	return vp.SetupVPRoutes(config.BaseRouter, vpConfig)
}