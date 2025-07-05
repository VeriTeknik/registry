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

// ServerFilterOptions represents filtering options for server queries
type ServerFilterOptions struct {
	Name            string
	RepositoryURL   string
	RepositorySource string
	Version         string
	IsLatest        *bool
	PackageRegistry string
}

// ConsolidatedServerFilter provides optimized server filtering with minimal database queries
type ConsolidatedServerFilter struct {
	registry service.RegistryService
	database database.Database // Direct database access for optimization
}

// NewConsolidatedServerFilter creates a new server filter instance
func NewConsolidatedServerFilter(registry service.RegistryService, db database.Database) *ConsolidatedServerFilter {
	return &ConsolidatedServerFilter{
		registry: registry,
		database: db,
	}
}

// ListWithFilters provides efficient server filtering with proper database queries
func (f *ConsolidatedServerFilter) ListWithFilters(filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	// Separate package registry filter which requires special handling
	packageRegistry, hasPackageFilter := filters["package_registry"].(string)
	if hasPackageFilter {
		delete(filters, "package_registry")
	}

	// Use direct database filtering if available
	if f.database != nil && !hasPackageFilter {
		return f.listWithDatabaseFilters(filters, cursor, limit)
	}

	// Fall back to memory filtering for complex queries
	return f.listWithMemoryFilters(filters, packageRegistry, hasPackageFilter, cursor, limit)
}

// listWithDatabaseFilters uses database-level filtering for efficiency
func (f *ConsolidatedServerFilter) listWithDatabaseFilters(filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use direct database filtering
	entries, nextCursor, err := f.database.List(ctx, filters, cursor, limit)
	if err != nil {
		// Fall back to memory filtering if database filtering fails
		log.Printf("Database filtering failed, falling back to memory filtering: %v", err)
		return f.listWithMemoryFilters(filters, "", false, cursor, limit)
	}

	// Convert []*model.Server to []model.Server
	result := make([]model.Server, len(entries))
	for i, entry := range entries {
		result[i] = *entry
	}

	return result, nextCursor, nil
}

// listWithMemoryFilters handles complex filtering that requires loading data into memory
func (f *ConsolidatedServerFilter) listWithMemoryFilters(filters map[string]interface{}, packageRegistry string, hasPackageFilter bool, cursor string, limit int) ([]model.Server, string, error) {
	// Strategy: For package registry filters or when database filtering fails,
	// we need to load servers in batches and filter in memory
	
	var allServers []model.Server
	currentCursor := cursor
	batchSize := 100
	maxServers := limit * 5 // Load up to 5x the requested limit to find matches

	// If we have a package registry filter, we need to be more aggressive
	if hasPackageFilter {
		maxServers = 1000 // Package filtering requires checking server details
	}

	for len(allServers) < maxServers {
		batch, nextCursor, err := f.registry.List(currentCursor, batchSize)
		if err != nil {
			return nil, "", err
		}

		// Apply basic filters to this batch
		filteredBatch := f.applyBasicFilters(batch, filters)

		// Apply package registry filter if needed
		if hasPackageFilter {
			filteredBatch = f.applyPackageRegistryFilter(filteredBatch, packageRegistry)
		}

		allServers = append(allServers, filteredBatch...)

		// Stop if we have enough results or no more data
		if len(allServers) >= limit || nextCursor == "" || len(batch) == 0 {
			break
		}

		currentCursor = nextCursor
	}

	// Apply cursor-based pagination to filtered results
	start := 0
	if cursor != "" && !hasPackageFilter {
		// Simple cursor implementation for basic filters
		for i, s := range allServers {
			if s.ID == cursor {
				start = i + 1
				break
			}
		}
	}

	end := start + limit
	if end > len(allServers) {
		end = len(allServers)
	}

	result := allServers[start:end]

	// Determine next cursor
	nextCursor := ""
	if end < len(allServers) && len(result) > 0 {
		nextCursor = result[len(result)-1].ID
	}

	log.Printf("ConsolidatedServerFilter: Found %d servers after filtering %d total servers", len(result), len(allServers))

	return result, nextCursor, nil
}

// applyBasicFilters applies simple field-based filters
func (f *ConsolidatedServerFilter) applyBasicFilters(servers []model.Server, filters map[string]interface{}) []model.Server {
	if len(filters) == 0 {
		return servers
	}

	var result []model.Server
	for _, server := range servers {
		if f.matchesBasicFilters(server, filters) {
			result = append(result, server)
		}
	}

	return result
}

// matchesBasicFilters checks if a server matches basic field filters
func (f *ConsolidatedServerFilter) matchesBasicFilters(server model.Server, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "name":
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

// applyPackageRegistryFilter filters servers by package registry
func (f *ConsolidatedServerFilter) applyPackageRegistryFilter(servers []model.Server, packageRegistry string) []model.Server {
	var result []model.Server
	
	for _, server := range servers {
		// Get full server details to check packages
		details, err := f.registry.GetByID(server.ID)
		if err != nil {
			log.Printf("Failed to get details for server %s: %v", server.ID, err)
			continue
		}

		// Check if any package matches the registry
		for _, pkg := range details.Packages {
			if pkg.RegistryName == packageRegistry {
				result = append(result, server)
				break
			}
		}
	}

	return result
}

// GetServersWithPackageRegistry provides an optimized way to get servers by package registry
func (f *ConsolidatedServerFilter) GetServersWithPackageRegistry(packageRegistry string, limit int) ([]model.Server, error) {
	filters := map[string]interface{}{
		"package_registry": packageRegistry,
	}
	
	servers, _, err := f.ListWithFilters(filters, "", limit)
	return servers, err
}

// Legacy compatibility functions

// ListWithFilters provides backward compatibility with the original function
func ListWithFilters(registry service.RegistryService, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	filter := &ConsolidatedServerFilter{registry: registry}
	return filter.listWithMemoryFilters(filters, "", false, cursor, limit)
}

// ListWithFiltersV2 provides backward compatibility with the V2 function
func ListWithFiltersV2(registry service.RegistryService, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	filter := &ConsolidatedServerFilter{registry: registry}
	return filter.ListWithFilters(filters, cursor, limit)
}

// DirectDatabaseListWithFilters provides direct database access for filtering
func DirectDatabaseListWithFilters(db database.Database, filters map[string]interface{}, cursor string, limit int) ([]model.Server, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entries, nextCursor, err := db.List(ctx, filters, cursor, limit)
	if err != nil {
		return nil, "", err
	}

	result := make([]model.Server, len(entries))
	for i, entry := range entries {
		result[i] = *entry
	}

	return result, nextCursor, nil
}