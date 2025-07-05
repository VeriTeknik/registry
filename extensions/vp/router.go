package vp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	Service          service.RegistryService
	MongoClient      *mongo.Client
	DatabaseName     string
	CacheTTL         time.Duration
	AnalyticsBaseURL string
}

// SetupVPRoutes sets up all VP (v-plugged) routes
func SetupVPRoutes(mux *http.ServeMux, config Config) error {
	log.Printf("Setting up VP routes with database: %s", config.DatabaseName)
	
	// Initialize stats database
	statsDB, err := stats.NewMongoDatabase(config.MongoClient, config.DatabaseName)
	if err != nil {
		return fmt.Errorf("failed to initialize stats database: %w", err)
	}
	log.Println("Stats database initialized")

	// Run migration to add source field to existing stats
	if err := statsDB.MigrateExistingStats(context.Background()); err != nil {
		// Log error but don't fail startup
		fmt.Printf("Warning: Failed to migrate existing stats: %v\n", err)
	}

	// Initialize feedback database
	log.Printf("Initializing feedback database with database name: %s", config.DatabaseName)
	feedbackDB, err := stats.NewMongoFeedbackDatabase(config.MongoClient, config.DatabaseName)
	if err != nil {
		return fmt.Errorf("failed to initialize feedback database: %w", err)
	}
	log.Println("Feedback database initialized successfully")

	// Initialize cache service
	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}
	cacheService := stats.NewCacheService(cacheTTL)

	// Initialize handlers
	vpHandlers := handlers.NewVPHandlers(config.Service, statsDB, feedbackDB, cacheService)

	// Server endpoints with stats
	mux.HandleFunc("/vp/servers", vpHandlers.GetServersHandler)
	
	// Register feedback endpoints with specific patterns
	// Pattern matching is more specific to avoid conflicts
	mux.HandleFunc("/vp/servers/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Extract path segments
		segments := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
		
		// Debug logging for ALL requests
		log.Printf("VP Route Debug - Method: %s, Path: %s, Segments: %v, Len: %d", r.Method, path, segments, len(segments))
		
		// Handle feedback endpoints first (more specific)
		if len(segments) >= 2 && segments[1] == "feedback" {
			switch r.Method {
			case http.MethodGet:
				vpHandlers.GetServerFeedbackHandler(w, r)
				return
			case http.MethodPost:
				// This would be for creating feedback, but we use /rate endpoint
				http.Error(w, "Use POST /vp/servers/{id}/rate to submit feedback", http.StatusBadRequest)
				return
			}
		}
		
		// Handle feedback update/delete: /vp/servers/{id}/feedback/{feedback_id}
		if len(segments) >= 3 && segments[1] == "feedback" {
			switch r.Method {
			case http.MethodPut:
				vpHandlers.UpdateFeedbackHandler(w, r)
				return
			case http.MethodDelete:
				vpHandlers.DeleteFeedbackHandler(w, r)
				return
			}
		}
		
		// Handle user rating check: /vp/servers/{id}/rating/{user_id}
		if len(segments) >= 3 && segments[1] == "rating" {
			if r.Method == http.MethodGet {
				vpHandlers.GetUserFeedbackHandler(w, r)
				return
			}
		}
		
		// Original routing logic - BUT CHECK SERVER DETAIL LAST
		// because it matches any single segment path
		switch {
		case r.Method == http.MethodGet && path == "/vp/servers/":
			vpHandlers.GetServersHandler(w, r)
		case r.Method == http.MethodPost && isInstallPath(path):
			vpHandlers.TrackInstallHandler(w, r)
		case r.Method == http.MethodPost && isRatePath(path):
			// Use feedback handler which supports comments
			vpHandlers.SubmitFeedbackHandler(w, r)
		case r.Method == http.MethodGet && isStatsPath(path):
			vpHandlers.GetStatsHandler(w, r)
		case r.Method == http.MethodPost && isClaimPath(path):
			vpHandlers.ClaimServerHandler(w, r)
		case r.Method == http.MethodGet && isServerDetailPath(path):
			// This MUST be last because it matches any single segment
			vpHandlers.GetServerByIDHandler(w, r)
		default:
			http.NotFound(w, r)
		}
	})


	// Global stats endpoints
	mux.HandleFunc("/vp/stats/global", vpHandlers.GetGlobalStatsHandler)
	mux.HandleFunc("/vp/stats/leaderboard", vpHandlers.GetLeaderboardHandler)
	mux.HandleFunc("/vp/stats/trending", vpHandlers.GetTrendingHandler)
	
	// Recent servers endpoints
	mux.HandleFunc("/vp/servers/recent", vpHandlers.GetRecentServersHandler)
	mux.HandleFunc("/vp/admin/timeline", vpHandlers.GetServerTimelineHandler)
	
	// Register feedback endpoints separately to ensure they work
	// This is a temporary workaround for the routing issue
	mux.HandleFunc("/vp/feedback/", func(w http.ResponseWriter, r *http.Request) {
		// Simple feedback endpoint that returns empty data for now
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"feedback": []interface{}{},
			"total_count": 0,
			"has_more": false,
		})
	})

	// Claim verification endpoint
	mux.HandleFunc("/vp/claim/verify", vpHandlers.GenerateClaimVerificationHandler)

	// Start analytics sync service if configured
	if config.AnalyticsBaseURL != "" {
		analyticsClient := stats.NewHTTPAnalyticsClient(config.AnalyticsBaseURL)
		syncService := stats.NewSyncService(statsDB, analyticsClient, 15*time.Minute)
		go syncService.Start(context.Background())
	}

	log.Println("VP routes setup completed successfully")
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