package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/models"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/service"
)

// TrackEvent handles event tracking
func TrackEvent(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.TrackEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		event := models.Event{
			Timestamp:  time.Now(),
			EventType:  req.EventType,
			ServerID:   req.ServerID,
			ClientID:   req.ClientID,
			SessionID:  req.SessionID,
			UserID:     req.UserID,
			Metadata:   req.Metadata,
		}

		if err := svc.TrackEvent(c.Request.Context(), &event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track event"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
	}
}

// GetServerStats returns server statistics
func GetServerStats(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		
		stats, err := svc.GetServerStats(c.Request.Context(), serverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

// GetServerTimeline returns time-series data for a server
func GetServerTimeline(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		period := c.DefaultQuery("period", "30d")
		
		timeline, err := svc.GetServerTimeline(c.Request.Context(), serverID, period)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"server_id": serverID,
			"period": period,
			"timeline": timeline,
		})
	}
}

// GetTrending returns trending servers
func GetTrending(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "24h")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		
		trending, err := svc.GetTrending(c.Request.Context(), period, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trending"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"period": period,
			"servers": trending,
		})
	}
}

// GetPopular returns popular servers
func GetPopular(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.DefaultQuery("category", "all")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		
		popular, err := svc.GetPopular(c.Request.Context(), category, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"category": category,
			"servers": popular,
		})
	}
}

// SearchServers handles server search
func SearchServers(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.SearchRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set defaults
		if req.Limit == 0 {
			req.Limit = 20
		}

		results, err := svc.SearchServers(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
			return
		}

		c.JSON(http.StatusOK, results)
	}
}

// RateServer handles server ratings
func RateServer(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		
		// TODO: Extract user ID from authentication
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		var req struct {
			Rating  int    `json:"rating" binding:"required,min=1,max=5"`
			Comment string `json:"comment"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rating := models.Rating{
			ServerID:  serverID,
			UserID:    userID,
			Rating:    req.Rating,
			Comment:   req.Comment,
			Timestamp: time.Now(),
		}

		if err := svc.RateServer(c.Request.Context(), &rating); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save rating"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "created"})
	}
}

// GetRatings returns server ratings
func GetRatings(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		
		ratings, total, err := svc.GetRatings(c.Request.Context(), serverID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ratings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ratings": ratings,
			"total": total,
			"limit": limit,
			"offset": offset,
		})
	}
}

// CommentOnServer handles comments
func CommentOnServer(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		
		// TODO: Extract user ID from authentication
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		var req struct {
			Comment  string `json:"comment" binding:"required,min=1,max=1000"`
			ParentID string `json:"parent_id"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		comment := models.Comment{
			ServerID:  serverID,
			UserID:    userID,
			Comment:   req.Comment,
			ParentID:  req.ParentID,
			Timestamp: time.Now(),
		}

		id, err := svc.AddComment(c.Request.Context(), &comment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save comment"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id": id,
			"status": "created",
		})
	}
}

// GetComments returns server comments
func GetComments(svc *service.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverID := c.Param("id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		
		comments, total, err := svc.GetComments(c.Request.Context(), serverID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"comments": comments,
			"total": total,
			"limit": limit,
			"offset": offset,
		})
	}
}

// HealthCheck returns service health status
func HealthCheck(esClient interface{}, redisClient interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement actual health checks
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"elasticsearch": "connected",
			"redis": "connected",
			"timestamp": time.Now(),
		})
	}
}