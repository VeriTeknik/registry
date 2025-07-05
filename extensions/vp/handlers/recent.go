package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetRecentServersHandler returns servers ordered by when they were first seen
func (h *VPHandlers) GetRecentServersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity && source != "ALL" {
		http.Error(w, "Invalid source. Must be 'REGISTRY', 'COMMUNITY', or 'ALL'", http.StatusBadRequest)
		return
	}

	// Check if we want to get servers from the last N days
	daysStr := r.URL.Query().Get("days")
	var sinceTime *time.Time
	if daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			since := time.Now().AddDate(0, 0, -days)
			sinceTime = &since
		}
	}

	// Check cache
	cacheKey := fmt.Sprintf("vp:recent:%s:%d:%s", source, limit, daysStr)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get recent servers from stats
	recentStats, err := h.statsDB.GetRecentServers(r.Context(), limit, source)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent servers: %v", err), http.StatusInternalServerError)
		return
	}

	// If we need to filter by days, do it here
	if sinceTime != nil {
		filtered := make([]*stats.ServerStats, 0)
		for _, stat := range recentStats {
			if stat.FirstSeen.After(*sinceTime) {
				filtered = append(filtered, stat)
			}
		}
		recentStats = filtered
	}

	// Get server details for each recent server
	servers := make([]*model.Server, 0, len(recentStats))
	statsMap := make(map[string]*stats.ServerStats)

	for _, stat := range recentStats {
		statsMap[stat.ServerID] = stat

		// Get server details
		serverDetail, err := h.service.GetByID(stat.ServerID)
		if err != nil {
			// Skip servers that can't be found
			continue
		}
		servers = append(servers, &serverDetail.Server)
	}

	// Create extended servers response
	extendedServers := vpmodel.NewExtendedServers(servers, statsMap)

	// Add first_seen info to response
	type RecentServerResponse struct {
		*vpmodel.ExtendedServer
		FirstSeen time.Time `json:"first_seen"`
		Source    string    `json:"discovered_via"`
	}

	recentServers := make([]RecentServerResponse, 0, len(extendedServers))
	for i := range extendedServers {
		es := &extendedServers[i]
		if stat, ok := statsMap[es.ID]; ok {
			recentServers = append(recentServers, RecentServerResponse{
				ExtendedServer: es,
				FirstSeen:      stat.FirstSeen,
				Source:         "stats", // Discovered via user interaction
			})
		}
	}

	// If we don't have enough from stats, check for recently imported servers
	if len(recentServers) < limit && source != stats.SourceCommunity {
		// Get recently added servers from the main collection using ObjectId timestamp
		additionalServers, err := h.getRecentlyImportedServers(r.Context(), limit-len(recentServers), sinceTime)
		if err == nil {
			for _, server := range additionalServers {
				// Skip if already in results
				found := false
				for _, rs := range recentServers {
					if rs.ID == server.ID {
						found = true
						break
					}
				}
				if !found {
					// Get stats if available
					serverStats, _ := h.statsDB.GetStats(r.Context(), server.ID, stats.SourceRegistry)
					extServer := vpmodel.NewExtendedServer(server, serverStats)
					
					// Extract creation time from ObjectId
					objId, _ := primitive.ObjectIDFromHex(server.ID)
					firstSeen := objId.Timestamp()
					
					recentServers = append(recentServers, RecentServerResponse{
						ExtendedServer: &extServer,
						FirstSeen:      firstSeen,
						Source:         "import", // Added via import/seed
					})
				}
			}
		}
	}

	response := map[string]interface{}{
		"servers":     recentServers,
		"total_count": len(recentServers),
		"filter": map[string]interface{}{
			"source": source,
			"limit":  limit,
			"days":   daysStr,
		},
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

// getRecentlyImportedServers gets servers from the main collection ordered by ObjectId (creation time)
func (h *VPHandlers) getRecentlyImportedServers(ctx context.Context, limit int, since *time.Time) ([]*model.Server, error) {
	// This would need access to the MongoDB client directly
	// For now, return empty - this is a placeholder for future enhancement
	return []*model.Server{}, nil
}

// GetServerTimelineHandler returns a timeline of server additions
func (h *VPHandlers) GetServerTimelineHandler(w http.ResponseWriter, r *http.Request) {
	// Get timeline data grouped by day/week/month
	period := r.URL.Query().Get("period") // day, week, month
	if period == "" {
		period = "day"
	}

	days := 30 // Default to last 30 days
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	// For now, return a simple message
	// This would be enhanced to show timeline data
	response := map[string]interface{}{
		"message": "Timeline endpoint - coming soon",
		"period":  period,
		"days":    days,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}