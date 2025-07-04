package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	"github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/types"
)

// TrackInstallHandler tracks an installation for a server
func (h *VPHandlers) TrackInstallHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Parse optional install request body
	var installReq stats.InstallRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&installReq); err != nil {
			// It's okay if body is empty or invalid, we just track the count
		}
	}

	// Increment installation count
	if err := h.statsDB.IncrementInstallCount(r.Context(), serverID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to track installation: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache for this server
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete("vp:servers:") // Clear servers list cache

	// TODO: Send install event to analytics service
	// This would track more detailed metrics like platform, version, etc.

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Installation tracked successfully",
	})
}

// SubmitRatingHandler submits a rating for a server
func (h *VPHandlers) SubmitRatingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Parse rating request
	var ratingReq stats.RatingRequest
	if err := json.NewDecoder(r.Body).Decode(&ratingReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate rating
	if ratingReq.Rating < 1 || ratingReq.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Update rating
	if err := h.statsDB.UpdateRating(r.Context(), serverID, ratingReq.Rating); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit rating: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete("vp:servers:") // Clear servers list cache

	// Return updated stats
	updatedStats, err := h.statsDB.GetStats(r.Context(), serverID)
	if err != nil {
		// Still return success even if we can't get updated stats
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Rating submitted successfully",
		})
		return
	}

	// Return success with updated stats
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rating submitted successfully",
		"stats":   updatedStats,
	})
}

// GetStatsHandler returns stats for a specific server
func (h *VPHandlers) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Check cache
	cacheKey := fmt.Sprintf("vp:stats:%s", serverID)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get stats
	serverStats, err := h.statsDB.GetStats(r.Context(), serverID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	response := stats.StatsResponse{
		Stats: serverStats,
	}

	// Cache the response
	h.statsCache.Set(cacheKey, response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetGlobalStatsHandler returns global registry statistics
func (h *VPHandlers) GetGlobalStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Check cache
	cacheKey := "vp:stats:global"
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get global stats
	globalStats, err := h.statsDB.GetGlobalStats(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get global stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the response
	h.statsCache.Set(cacheKey, globalStats)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(globalStats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetLeaderboardHandler returns leaderboard data
func (h *VPHandlers) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get leaderboard type from query params
	leaderboardType := r.URL.Query().Get("type")
	if leaderboardType == "" {
		leaderboardType = string(stats.LeaderboardTypeInstalls)
	}

	// Get limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Check cache
	cacheKey := fmt.Sprintf("vp:leaderboard:%s:%d", leaderboardType, limit)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	var leaderboardData interface{}
	var err error

	switch stats.LeaderboardType(leaderboardType) {
	case stats.LeaderboardTypeInstalls:
		leaderboardData, err = h.statsDB.GetTopByInstalls(r.Context(), limit)
	case stats.LeaderboardTypeRating:
		leaderboardData, err = h.statsDB.GetTopByRating(r.Context(), limit)
	case stats.LeaderboardTypeTrending:
		leaderboardData, err = h.statsDB.GetTrending(r.Context(), limit)
	default:
		http.Error(w, "Invalid leaderboard type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"type":  leaderboardType,
		"limit": limit,
		"data":  leaderboardData,
	}

	// Cache the response
	h.statsCache.Set(cacheKey, response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetTrendingHandler returns trending servers
func (h *VPHandlers) GetTrendingHandler(w http.ResponseWriter, r *http.Request) {
	// Get limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Check cache
	cacheKey := fmt.Sprintf("vp:trending:%d", limit)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get trending servers
	trendingServers, err := h.statsDB.GetTrending(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get trending servers: %v", err), http.StatusInternalServerError)
		return
	}

	// Get full server details for trending servers
	serverIDs := make([]string, len(trendingServers))
	for i, stats := range trendingServers {
		serverIDs[i] = stats.ServerID
	}

	// Get servers from main service
	servers := make([]*types.Server, 0)
	for _, id := range serverIDs {
		server, err := h.service.GetServerByID(r.Context(), id)
		if err == nil {
			servers = append(servers, server)
		}
	}

	// Create stats map
	statsMap := make(map[string]*stats.ServerStats)
	for _, s := range trendingServers {
		statsMap[s.ServerID] = s
	}

	// Create extended servers response
	extendedServers := model.NewExtendedServers(servers, statsMap)
	response := map[string]interface{}{
		"limit":   limit,
		"servers": extendedServers,
	}

	// Cache the response
	h.statsCache.Set(cacheKey, response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}