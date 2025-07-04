package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// VPHandlers contains the handlers for VP (v-plugged) endpoints
type VPHandlers struct {
	service      service.RegistryService
	statsDB      stats.Database
	statsCache   *stats.CacheService
}

// NewVPHandlers creates a new instance of VPHandlers
func NewVPHandlers(service service.RegistryService, statsDB stats.Database, statsCache *stats.CacheService) *VPHandlers {
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

	// Get stats for all servers
	statsMap, err := h.statsDB.GetBatchStats(r.Context(), serverIDs)
	if err != nil {
		// Log error but continue without stats
		fmt.Printf("Failed to get stats: %v\n", err)
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

	// Get stats for the server
	serverStats, err := h.statsDB.GetStats(r.Context(), serverID)
	if err != nil {
		// Log error but continue without stats
		fmt.Printf("Failed to get stats for server %s: %v\n", serverID, err)
		serverStats = &stats.ServerStats{ServerID: serverID}
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