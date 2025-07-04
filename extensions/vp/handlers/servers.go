package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	"github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/service"
	"github.com/modelcontextprotocol/registry/internal/types"
)

// VPHandlers contains the handlers for VP (v-plugged) endpoints
type VPHandlers struct {
	service      *service.Service
	statsDB      stats.Database
	statsCache   *stats.CacheService
}

// NewVPHandlers creates a new instance of VPHandlers
func NewVPHandlers(service *service.Service, statsDB stats.Database, statsCache *stats.CacheService) *VPHandlers {
	return &VPHandlers{
		service:    service,
		statsDB:    statsDB,
		statsCache: statsCache,
	}
}

// GetServersHandler returns a list of servers with stats included
func (h *VPHandlers) GetServersHandler(w http.ResponseWriter, r *http.Request) {
	// Check cache first
	cacheKey := "vp:servers:" + r.URL.Query().Encode()
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get servers from the main service
	servers, err := h.service.GetServers(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}

	// Get server IDs for batch stats lookup
	serverIDs := make([]string, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	// Get stats for all servers
	statsMap, err := h.statsDB.GetBatchStats(r.Context(), serverIDs)
	if err != nil {
		// Log error but continue without stats
		fmt.Printf("Failed to get stats: %v\n", err)
		statsMap = make(map[string]*stats.ServerStats)
	}

	// Create extended servers response
	extendedServers := model.NewExtendedServers(servers, statsMap)
	response := model.ExtendedServersResponse{
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
	server, err := h.service.GetServerByID(r.Context(), serverID)
	if err != nil {
		if err == service.ErrServerNotFound {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get server: %v", err), http.StatusInternalServerError)
		return
	}

	// Get stats for the server
	serverStats, err := h.statsDB.GetStats(r.Context(), serverID)
	if err != nil {
		// Log error but continue without stats
		fmt.Printf("Failed to get stats for server %s: %v\n", serverID, err)
		serverStats = &stats.ServerStats{ServerID: serverID}
	}

	// Create extended server response
	extendedServer := model.NewExtendedServer(server, serverStats)
	response := model.ExtendedServerResponse{
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

// SearchServersHandler searches servers with stats included
func (h *VPHandlers) SearchServersHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	// For now, redirect to regular search and enhance with stats
	// In a real implementation, this would integrate with the search service
	servers, err := h.service.SearchServers(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Get server IDs for batch stats lookup
	serverIDs := make([]string, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	// Get stats for all servers
	statsMap, err := h.statsDB.GetBatchStats(r.Context(), serverIDs)
	if err != nil {
		statsMap = make(map[string]*stats.ServerStats)
	}

	// Create extended servers response
	extendedServers := model.NewExtendedServers(servers, statsMap)
	response := model.ExtendedServersResponse{
		Servers: extendedServers,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Helper method to convert basic servers to type Server pointers
func convertToServerPointers(servers []types.Server) []*types.Server {
	result := make([]*types.Server, len(servers))
	for i := range servers {
		result[i] = &servers[i]
	}
	return result
}