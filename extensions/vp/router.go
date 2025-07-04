package vp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	"github.com/modelcontextprotocol/registry/extensions/vp/handlers"
	"github.com/modelcontextprotocol/registry/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
)

// Config holds configuration for VP endpoints
type Config struct {
	Service          *service.Service
	MongoClient      *mongo.Client
	DatabaseName     string
	CacheTTL         time.Duration
	AnalyticsBaseURL string
}

// SetupVPRoutes sets up all VP (v-plugged) routes
func SetupVPRoutes(mux *http.ServeMux, config Config) error {
	// Initialize stats database
	statsDB, err := stats.NewMongoDatabase(config.MongoClient, config.DatabaseName)
	if err != nil {
		return err
	}

	// Initialize cache service
	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}
	cacheService := stats.NewCacheService(cacheTTL)

	// Initialize handlers
	vpHandlers := handlers.NewVPHandlers(config.Service, statsDB, cacheService)

	// Server endpoints with stats
	mux.HandleFunc("/vp/servers", vpHandlers.GetServersHandler)
	mux.HandleFunc("/vp/servers/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && path == "/vp/servers/":
			vpHandlers.GetServersHandler(w, r)
		case r.Method == http.MethodGet && isServerDetailPath(path):
			vpHandlers.GetServerByIDHandler(w, r)
		case r.Method == http.MethodPost && isInstallPath(path):
			vpHandlers.TrackInstallHandler(w, r)
		case r.Method == http.MethodPost && isRatePath(path):
			vpHandlers.SubmitRatingHandler(w, r)
		case r.Method == http.MethodGet && isStatsPath(path):
			vpHandlers.GetStatsHandler(w, r)
		case r.Method == http.MethodPost && isClaimPath(path):
			vpHandlers.ClaimServerHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	// Search endpoint
	mux.HandleFunc("/vp/search", vpHandlers.SearchServersHandler)

	// Global stats endpoints
	mux.HandleFunc("/vp/stats/global", vpHandlers.GetGlobalStatsHandler)
	mux.HandleFunc("/vp/stats/leaderboard", vpHandlers.GetLeaderboardHandler)
	mux.HandleFunc("/vp/stats/trending", vpHandlers.GetTrendingHandler)

	// Claim verification endpoint
	mux.HandleFunc("/vp/claim/verify", vpHandlers.GenerateClaimVerificationHandler)

	// Start analytics sync service if configured
	if config.AnalyticsBaseURL != "" {
		analyticsClient := stats.NewHTTPAnalyticsClient(config.AnalyticsBaseURL)
		syncService := stats.NewSyncService(statsDB, analyticsClient, 15*time.Minute)
		go syncService.Start(context.Background())
	}

	return nil
}

// Path helper functions
func isServerDetailPath(path string) bool {
	// /vp/servers/{id} (no trailing segments)
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	return len(parts) == 1 && parts[0] != ""
}

func isInstallPath(path string) bool {
	return strings.Contains(path, "/install")
}

func isRatePath(path string) bool {
	return strings.Contains(path, "/rate")
}

func isStatsPath(path string) bool {
	return strings.Contains(path, "/stats") && !strings.Contains(path, "/stats/")
}

func isClaimPath(path string) bool {
	return strings.Contains(path, "/claim")
}