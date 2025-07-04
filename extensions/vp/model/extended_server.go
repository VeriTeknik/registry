package model

import (
	"github.com/modelcontextprotocol/registry/internal/types"
	"github.com/modelcontextprotocol/registry/extensions/stats"
)

// ExtendedServer represents a server with stats included
type ExtendedServer struct {
	*types.Server
	InstallationCount  int     `json:"installation_count"`
	Rating            float64 `json:"rating"`
	RatingCount       int     `json:"rating_count"`
	ActiveInstalls    int     `json:"active_installs,omitempty"`
	WeeklyGrowth      float64 `json:"weekly_growth,omitempty"`
}

// ExtendedServersResponse represents the response for listing servers with stats
type ExtendedServersResponse struct {
	Servers []ExtendedServer `json:"servers"`
}

// ExtendedServerResponse represents the response for a single server with stats
type ExtendedServerResponse struct {
	Server ExtendedServer `json:"server"`
}

// NewExtendedServer creates an ExtendedServer from a Server and ServerStats
func NewExtendedServer(server *types.Server, stats *stats.ServerStats) ExtendedServer {
	es := ExtendedServer{
		Server: server,
	}
	
	if stats != nil {
		es.InstallationCount = stats.InstallationCount
		es.Rating = stats.Rating
		es.RatingCount = stats.RatingCount
		es.ActiveInstalls = stats.ActiveInstalls
		// Weekly growth would be calculated by the analytics service
	}
	
	return es
}

// NewExtendedServers creates a slice of ExtendedServers from servers and their stats
func NewExtendedServers(servers []*types.Server, statsMap map[string]*stats.ServerStats) []ExtendedServer {
	result := make([]ExtendedServer, 0, len(servers))
	
	for _, server := range servers {
		stats := statsMap[server.ID]
		result = append(result, NewExtendedServer(server, stats))
	}
	
	return result
}