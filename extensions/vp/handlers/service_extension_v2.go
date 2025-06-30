package handlers

import (
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// ListWithFiltersV2 extends the registry service to support filtering including package registry
func ListWithFiltersV2(registry service.RegistryService, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	// For package_registry filter, we need to fetch details for each server
	packageRegistry, hasPackageFilter := filters["package_registry"].(string)
	
	// Remove package_registry from filters since it's not a direct server field
	if hasPackageFilter {
		delete(filters, "package_registry")
	}
	
	// Get servers with basic filters
	servers, nextCursor, err := ListWithFilters(registry, filters, cursor, limit)
	if err != nil {
		return nil, "", err
	}
	
	// If no package filter, return as-is
	if !hasPackageFilter {
		return servers, nextCursor, nil
	}
	
	// Filter by package registry
	var filtered []model.Server
	for _, server := range servers {
		// Get full server details to check packages
		details, err := registry.GetByID(server.ID)
		if err != nil {
			continue // Skip if can't get details
		}
		
		// Check if any package matches the registry
		for _, pkg := range details.Packages {
			if pkg.RegistryName == packageRegistry {
				filtered = append(filtered, server)
				break
			}
		}
	}
	
	return filtered, nextCursor, nil
}

// GetServersWithPackageRegistry fetches all servers that have packages in a specific registry
func GetServersWithPackageRegistry(registry service.RegistryService, packageRegistry string, limit int) ([]model.Server, error) {
	var allServers []model.Server
	cursor := ""
	
	// Fetch all servers in batches
	for {
		servers, nextCursor, err := registry.List(cursor, 100) // Fetch 100 at a time
		if err != nil {
			return nil, err
		}
		
		// Check each server for matching package registry
		for _, server := range servers {
			details, err := registry.GetByID(server.ID)
			if err != nil {
				continue
			}
			
			for _, pkg := range details.Packages {
				if pkg.RegistryName == packageRegistry {
					allServers = append(allServers, server)
					break
				}
			}
			
			// Stop if we've found enough
			if len(allServers) >= limit {
				return allServers[:limit], nil
			}
		}
		
		// Check if there are more pages
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	
	return allServers, nil
}