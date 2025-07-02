package handlers

import (
	"context"
	"log"
	"strings"
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
	
	// If we have filters, we need to get ALL servers to filter properly
	// This is not ideal for large datasets but necessary for now
	if len(filters) > 0 {
		allServers := []model.Server{}
		currentCursor := ""
		
		// Fetch all servers in batches
		for {
			batch, nextCursor, err := registry.List(currentCursor, 100) // Get 100 at a time
			if err != nil {
				return nil, "", err
			}
			
			allServers = append(allServers, batch...)
			
			// If no more pages, break
			if nextCursor == "" || len(batch) == 0 {
				break
			}
			currentCursor = nextCursor
		}
		
		log.Printf("ListWithFilters: Got %d total servers for filtering", len(allServers))
		
		// Apply filters
		filtered := filterServers(allServers, filters)
		
		// Apply pagination to filtered results
		start := 0
		if cursor != "" {
			// Simple cursor implementation - in production you'd want something better
			for i, s := range filtered {
				if s.ID == cursor {
					start = i + 1
					break
				}
			}
		}
		
		end := start + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		
		result := filtered[start:end]
		
		// Determine next cursor
		nextCursor := ""
		if end < len(filtered) {
			nextCursor = filtered[end].ID
		}
		
		return result, nextCursor, nil
	}
	
	// No filters - use normal pagination
	servers, nextCursor, err := registry.List(cursor, limit)
	if err != nil {
		return nil, "", err
	}
	
	return servers, nextCursor, nil
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
	
	// Debug logging
	if nameFilter, hasName := filters["name"]; hasName {
		log.Printf("filterServers: name filter='%s', found %d matching servers out of %d total", nameFilter, len(result), len(servers))
	}
	
	return result
}

// matchesFilters checks if a server matches all the provided filters
func matchesFilters(server model.Server, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "name":
			// Support partial, case-insensitive name matching
			nameFilter := strings.ToLower(value.(string))
			serverName := strings.ToLower(server.Name)
			if !strings.Contains(serverName, nameFilter) {
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