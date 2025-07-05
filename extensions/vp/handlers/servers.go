package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/auth"
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// VPHandlers contains the handlers for VP (v-plugged) endpoints
type VPHandlers struct {
	service         service.RegistryService
	statsDB         stats.Database
	feedbackDB      stats.FeedbackDatabase
	analyticsDB     stats.AnalyticsDatabase
	analyticsClient stats.AnalyticsClient
	statsCache      *stats.CacheService
	authService     auth.Service
}

// NewVPHandlers creates a new instance of VPHandlers
func NewVPHandlers(service service.RegistryService, statsDB stats.Database, feedbackDB stats.FeedbackDatabase, analyticsDB stats.AnalyticsDatabase, analyticsClient stats.AnalyticsClient, statsCache *stats.CacheService, authService auth.Service) *VPHandlers {
	return &VPHandlers{
		service:         service,
		statsDB:         statsDB,
		feedbackDB:      feedbackDB,
		analyticsDB:     analyticsDB,
		analyticsClient: analyticsClient,
		statsCache:      statsCache,
		authService:     authService,
	}
}

// GetServersHandler returns a list of servers with stats included
func (h *VPHandlers) GetServersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	sortBy := r.URL.Query().Get("sort")
	source := r.URL.Query().Get("source")
	limitStr := r.URL.Query().Get("limit")
	
	// Validate source
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity {
		http.Error(w, "Invalid source. Must be 'REGISTRY' or 'COMMUNITY'", http.StatusBadRequest)
		return
	}

	// Parse limit
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	// Check cache first
	cacheKey := "vp:servers:" + r.URL.Query().Encode()
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Handle sorted requests differently
	if sortBy != "" {
		h.handleSortedServers(w, r, sortBy, source, limit, cacheKey)
		return
	}

	// Get servers from the main service
	// Using List with large limit to get all servers
	serverList, _, err := h.service.List("", 1000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert to pointers
	servers := make([]*model.Server, len(serverList))
	for i := range serverList {
		servers[i] = &serverList[i]
	}

	// Get server IDs for batch stats lookup
	serverIDs := make([]string, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	// Get stats for all servers with source if specified
	statsMap, err := h.statsDB.GetBatchStats(r.Context(), serverIDs, source)
	if err != nil {
		// Log error but continue without stats
		log.Printf("Failed to get stats: %v", err)
		statsMap = make(map[string]*stats.ServerStats)
	}

	// Create extended servers response
	extendedServers := vpmodel.NewExtendedServers(servers, statsMap)
	response := vpmodel.ExtendedServersResponse{
		Servers: extendedServers,
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

// GetServerByIDHandler returns a single server with stats
func (h *VPHandlers) GetServerByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from URL path
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Check cache
	cacheKey := fmt.Sprintf("vp:server:%s", serverID)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get server from main service
	serverDetail, err := h.service.GetByID(serverID)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}
	
	// ServerDetail already contains Server
	server := &serverDetail.Server

	// Get source from query params
	source := r.URL.Query().Get("source")
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity {
		http.Error(w, "Invalid source. Must be 'REGISTRY' or 'COMMUNITY'", http.StatusBadRequest)
		return
	}

	// Get stats for the server
	serverStats, err := h.statsDB.GetStats(r.Context(), serverID, source)
	if err != nil {
		// Log error but continue without stats
		log.Printf("Failed to get stats for server %s: %v", serverID, err)
		serverStats = &stats.ServerStats{ServerID: serverID, Source: source}
	}

	// Create extended server response
	extendedServer := vpmodel.NewExtendedServer(server, serverStats)
	response := vpmodel.ExtendedServerResponse{
		Server: extendedServer,
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


// Helper method to convert basic servers to type Server pointers
func convertToServerPointers(servers []model.Server) []*model.Server {
	result := make([]*model.Server, len(servers))
	for i := range servers {
		result[i] = &servers[i]
	}
	return result
}

// handleSortedServers handles server requests with sorting
func (h *VPHandlers) handleSortedServers(w http.ResponseWriter, r *http.Request, sortBy, source string, limit int, cacheKey string) {
	var sortedStats []*stats.ServerStats
	var err error

	// Get sorted stats based on sort parameter
	switch sortBy {
	case "installs":
		sortedStats, err = h.statsDB.GetTopByInstalls(r.Context(), limit, source)
	case "rating":
		sortedStats, err = h.statsDB.GetTopByRating(r.Context(), limit, source)
	case "trending":
		sortedStats, err = h.statsDB.GetTrending(r.Context(), limit, source)
	default:
		http.Error(w, "Invalid sort parameter. Must be 'installs', 'rating', or 'trending'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get sorted servers: %v", err), http.StatusInternalServerError)
		return
	}

	// Get server details for the sorted servers
	servers := make([]*model.Server, 0, len(sortedStats))
	statsMap := make(map[string]*stats.ServerStats)
	
	for _, stat := range sortedStats {
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

	// Sort extended servers to maintain the order from stats query
	sort.Slice(extendedServers, func(i, j int) bool {
		switch sortBy {
		case "installs":
			return extendedServers[i].InstallationCount > extendedServers[j].InstallationCount
		case "rating":
			return extendedServers[i].Rating > extendedServers[j].Rating
		case "trending":
			// Trending is already sorted by the database query
			return false
		default:
			return false
		}
	})

	response := vpmodel.ExtendedServersResponse{
		Servers: extendedServers,
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