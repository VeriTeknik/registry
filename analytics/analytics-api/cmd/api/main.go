package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/elasticsearch"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/handlers"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/redis"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/service"
)

func main() {
	log.Println("Starting Analytics API...")

	// Configuration
	esURL := getEnv("ELASTICSEARCH_URL", "http://localhost:9200")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	port := getEnv("PORT", "8081")
	corsOrigins := getEnv("CORS_ORIGINS", "http://localhost:3000")

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(esURL)
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}

	// Initialize Redis client
	redisClient, err := redis.NewClient(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create service
	analyticsService := service.NewAnalyticsService(esClient, redisClient)

	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Set up routes
	api := router.Group("/api/v1")
	{
		// Event tracking
		api.POST("/track", handlers.TrackEvent(analyticsService))
		
		// Server analytics
		api.GET("/servers/:id/stats", handlers.GetServerStats(analyticsService))
		api.GET("/servers/:id/timeline", handlers.GetServerTimeline(analyticsService))
		
		// Trending and popular
		api.GET("/trending", handlers.GetTrending(analyticsService))
		api.GET("/popular", handlers.GetPopular(analyticsService))
		
		// Search
		api.GET("/search", handlers.SearchServers(analyticsService))
		
		// Feedback
		api.POST("/servers/:id/rate", handlers.RateServer(analyticsService))
		api.GET("/servers/:id/ratings", handlers.GetRatings(analyticsService))
		api.POST("/servers/:id/comment", handlers.CommentOnServer(analyticsService))
		api.GET("/servers/:id/comments", handlers.GetComments(analyticsService))
		
		// Health check
		api.GET("/health", handlers.HealthCheck(esClient, redisClient))
	}

	// Add metrics endpoints (compatible with frontend expectations)
	metrics := router.Group("/api/metrics")
	{
		metrics.GET("/global", handlers.GetGlobalMetrics(analyticsService))
		metrics.GET("/trending", handlers.GetMetricsTrending(analyticsService))
	}

	// Add events endpoints (compatible with frontend expectations)
	events := router.Group("/api/events")
	{
		events.POST("/batch", handlers.BatchTrackEvents(analyticsService))
	}

	// CORS middleware
	origins := strings.Split(corsOrigins, ",")
	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           86400,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: c.Handler(router),
	}

	// Start server in goroutine
	go func() {
		log.Printf("Analytics API listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}