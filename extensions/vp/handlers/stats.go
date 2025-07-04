package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/model"
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

	// Validate source if provided
	if installReq.Source != "" && installReq.Source != stats.SourceRegistry && installReq.Source != stats.SourceCommunity {
		http.Error(w, "Invalid source. Must be 'REGISTRY' or 'COMMUNITY'", http.StatusBadRequest)
		return
	}

	// Default source to REGISTRY if not specified
	if installReq.Source == "" {
		installReq.Source = stats.SourceRegistry
	}

	// Increment installation count with source
	if err := h.statsDB.IncrementInstallCount(r.Context(), serverID, installReq.Source); err != nil {
		http.Error(w, fmt.Sprintf("Failed to track installation: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache for this server
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s:%s", serverID, installReq.Source))
	h.statsCache.Delete("vp:servers:") // Clear servers list cache
	h.statsCache.Delete("vp:stats:global") // Clear global stats cache
	h.statsCache.Delete(fmt.Sprintf("vp:stats:global:%s", installReq.Source)) // Clear source-specific global stats

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

// SubmitRatingHandler handles rating submission with backward compatibility
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

	// If user_id is provided and comment exists, use the feedback handler
	if ratingReq.UserID != "" || ratingReq.Comment != "" {
		// For feedback tracking, user_id is required
		if ratingReq.UserID == "" {
			// Generate a temporary user ID based on IP for anonymous ratings with comments
			ratingReq.UserID = "anon_" + strings.ReplaceAll(r.RemoteAddr, ":", "_")
		}
		
		// Rewind the request body
		bodyBytes, _ := json.Marshal(ratingReq)
		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		h.SubmitFeedbackHandler(w, r)
		return
	}

	// Otherwise, use the simple rating update (backward compatibility)
	// Default source to REGISTRY if not specified
	if ratingReq.Source == "" {
		ratingReq.Source = stats.SourceRegistry
	}

	// Update rating statistics
	if err := h.statsDB.UpdateRating(r.Context(), serverID, ratingReq.Source, ratingReq.Rating); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update rating: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s:%s", serverID, ratingReq.Source))
	h.statsCache.Delete("vp:servers:") // Clear servers list cache

	// Get updated stats
	updatedStats, err := h.statsDB.GetStats(r.Context(), serverID, ratingReq.Source)
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

	// Get source from query params (optional)
	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity {
		http.Error(w, "Invalid source. Must be 'REGISTRY' or 'COMMUNITY'", http.StatusBadRequest)
		return
	}

	// Check for aggregated request
	aggregated := r.URL.Query().Get("aggregated") == "true"

	// Build cache key
	cacheKey := fmt.Sprintf("vp:stats:%s", serverID)
	if source != "" {
		cacheKey = fmt.Sprintf("vp:stats:%s:%s", serverID, source)
	}
	if aggregated {
		cacheKey = fmt.Sprintf("vp:stats:%s:aggregated", serverID)
	}
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get stats based on request type
	var response interface{}
	if aggregated {
		// Get aggregated stats from all sources
		aggStats, err := h.statsDB.GetAggregatedStats(r.Context(), serverID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get aggregated stats: %v", err), http.StatusInternalServerError)
			return
		}
		response = aggStats
	} else {
		// Get stats for specific source
		serverStats, err := h.statsDB.GetStats(r.Context(), serverID, source)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
			return
		}
		response = stats.StatsResponse{
			Stats: serverStats,
		}
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
	// Get source from query params (optional)
	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity && source != "ALL" {
		http.Error(w, "Invalid source. Must be 'REGISTRY', 'COMMUNITY', or 'ALL'", http.StatusBadRequest)
		return
	}

	// Build cache key
	cacheKey := "vp:stats:global"
	if source != "" {
		cacheKey = fmt.Sprintf("vp:stats:global:%s", source)
	}
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get global stats
	globalStats, err := h.statsDB.GetGlobalStats(r.Context(), source)
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

	// Get source from query params (optional)
	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity && source != "ALL" {
		http.Error(w, "Invalid source. Must be 'REGISTRY', 'COMMUNITY', or 'ALL'", http.StatusBadRequest)
		return
	}

	// Build cache key
	cacheKey := fmt.Sprintf("vp:leaderboard:%s:%d", leaderboardType, limit)
	if source != "" {
		cacheKey = fmt.Sprintf("vp:leaderboard:%s:%d:%s", leaderboardType, limit, source)
	}
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
		leaderboardData, err = h.statsDB.GetTopByInstalls(r.Context(), limit, source)
	case stats.LeaderboardTypeRating:
		leaderboardData, err = h.statsDB.GetTopByRating(r.Context(), limit, source)
	case stats.LeaderboardTypeTrending:
		leaderboardData, err = h.statsDB.GetTrending(r.Context(), limit, source)
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

	// Get source from query params (optional)
	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity && source != "ALL" {
		http.Error(w, "Invalid source. Must be 'REGISTRY', 'COMMUNITY', or 'ALL'", http.StatusBadRequest)
		return
	}

	// Build cache key
	cacheKey := fmt.Sprintf("vp:trending:%d", limit)
	if source != "" {
		cacheKey = fmt.Sprintf("vp:trending:%d:%s", limit, source)
	}
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get trending servers
	trendingServers, err := h.statsDB.GetTrending(r.Context(), limit, source)
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
	servers := make([]*model.Server, 0)
	for _, id := range serverIDs {
		serverDetail, err := h.service.GetByID(id)
		if err == nil {
			server := &serverDetail.Server
			servers = append(servers, server)
		}
	}

	// Create stats map
	statsMap := make(map[string]*stats.ServerStats)
	for _, s := range trendingServers {
		statsMap[s.ServerID] = s
	}

	// Create extended servers response
	extendedServers := vpmodel.NewExtendedServers(servers, statsMap)
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