package handlers

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/registry/internal/database"
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// ListWithFilters extends the registry service to support filtering
// This is a wrapper that accesses the underlying database directly for filtering support
func ListWithFilters(registry service.RegistryService, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	// We need to access the database directly since the service layer doesn't expose filtering
	// This is a temporary solution until filtering is added to the service interface
	
	// For now, we'll use the registry's List method and filter in memory
	// In a production environment, you'd want to extend the service interface or create a new service
	
	// Get all results from the service (this is not ideal for large datasets)
	servers, nextCursor, err := registry.List(cursor, limit)
	if err != nil {
		return nil, "", err
	}
	
	// If no filters, return as-is
	if len(filters) == 0 {
		return servers, nextCursor, nil
	}
	
	// Apply filters in memory (temporary solution)
	filtered := filterServers(servers, filters)
	
	return filtered, nextCursor, nil
}

// filterServers applies filters to a list of servers in memory
func filterServers(servers []model.Server, filters map[string]interface{}) []model.Server {
	if len(filters) == 0 {
		return servers
	}
	
	var result []model.Server
	for _, server := range servers {
		if matchesFilters(server, filters) {
			result = append(result, server)
		}
	}
	
	return result
}

// matchesFilters checks if a server matches all the provided filters
func matchesFilters(server model.Server, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "name":
			if server.Name != value.(string) {
				return false
			}
		case "repository.url":
			if server.Repository.URL != value.(string) {
				return false
			}
		case "repository.source":
			if server.Repository.Source != value.(string) {
				return false
			}
		case "version":
			if server.VersionDetail.Version != value.(string) {
				return false
			}
		case "version_detail.is_latest":
			if server.VersionDetail.IsLatest != value.(bool) {
				return false
			}
		}
	}
	return true
}

// DirectDatabaseListWithFilters provides direct database access for filtering
// This requires access to the database instance
func DirectDatabaseListWithFilters(db database.Database, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Use the database's List method with filters
	entries, nextCursor, err := db.List(ctx, filters, cursor, limit)
	if err != nil {
		return nil, "", err
	}
	
	// Convert from []*model.Server to []model.Server
	result := make([]model.Server, len(entries))
	for i, entry := range entries {
		result[i] = *entry
	}
	
	return result, nextCursor, nil
}