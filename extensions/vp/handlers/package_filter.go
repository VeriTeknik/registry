package handlers

import (
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// FilterServersByPackageRegistry filters servers by package registry type
// This is a dedicated function for package registry filtering that fetches all servers
// and filters them properly
func FilterServersByPackageRegistry(registry service.RegistryService, packageRegistry string, limit int) ([]model.Server, error) {
	var filtered []model.Server
	cursor := ""
	
	// We need to check all servers since package info is only in details
	for len(filtered) < limit {
		// Get a batch of servers
		servers, nextCursor, err := registry.List(cursor, 100)
		if err != nil {
			return filtered, err
		}
		
		// Check each server's packages
		for _, server := range servers {
			if len(filtered) >= limit {
				break
			}
			
			// Get full details to check packages
			details, err := registry.GetByID(server.ID)
			if err != nil {
				continue
			}
			
			// Check if any package matches the registry
			hasMatchingPackage := false
			for _, pkg := range details.Packages {
				if pkg.RegistryName == packageRegistry {
					hasMatchingPackage = true
					break
				}
			}
			
			if hasMatchingPackage {
				filtered = append(filtered, server)
			}
		}
		
		// If no more pages, stop
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	
	return filtered, nil
}