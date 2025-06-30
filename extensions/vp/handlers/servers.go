package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/internal/service"
)

// PaginatedResponse is a paginated API response with filtering
type PaginatedResponse struct {
	Data     []model.Server `json:"servers"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

// Metadata contains pagination metadata
type Metadata struct {
	NextCursor string `json:"next_cursor,omitempty"`
	Count      int    `json:"count,omitempty"`
	Total      int    `json:"total,omitempty"`
}

// ServersHandler returns a handler for listing registry items with filtering support
func ServersHandler(registry service.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		queryParams := r.URL.Query()
		
		// Parse cursor and limit from query parameters
		cursor := queryParams.Get("cursor")
		if cursor != "" {
			_, err := uuid.Parse(cursor)
			if err != nil {
				http.Error(w, "Invalid cursor parameter", http.StatusBadRequest)
				return
			}
		}
		
		limitStr := queryParams.Get("limit")
		limit := 30 // Default limit
		
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
			
			if parsedLimit <= 0 {
				http.Error(w, "Limit must be greater than 0", http.StatusBadRequest)
				return
			}
			
			if parsedLimit > 100 {
				limit = 100 // Cap maximum limit
			} else {
				limit = parsedLimit
			}
		}

		// Build filter map from query parameters
		filters := buildFilters(r.URL.Query())

		var servers []model.Server
		var nextCursor string
		var err error

		// Special handling for package_registry filter
		if packageRegistry, ok := filters["package_registry"].(string); ok && packageRegistry != "" {
			// Use dedicated package registry filter
			servers, err = FilterServersByPackageRegistry(registry, packageRegistry, limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// No pagination cursor for package registry filtering yet
			nextCursor = ""
		} else {
			// Use regular filtering
			servers, nextCursor, err = ListWithFiltersV2(registry, filters, cursor, limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Create paginated response
		response := PaginatedResponse{
			Data: servers,
		}

		// Add metadata if there's a next cursor
		if nextCursor != "" {
			response.Metadata = Metadata{
				NextCursor: nextCursor,
				Count:      len(servers),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// ServersDetailHandler returns a handler for getting details of a specific server by ID
func ServersDetailHandler(registry service.RegistryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the server ID from the URL path
		id := r.PathValue("id")

		// Validate that the ID is a valid UUID
		_, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid server ID format", http.StatusBadRequest)
			return
		}

		// Get the server details from the registry service
		serverDetail, err := registry.GetByID(id)
		if err != nil {
			if err.Error() == "record not found" {
				http.Error(w, "Server not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Error retrieving server details", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(serverDetail); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// buildFilters constructs a filter map from query parameters
func buildFilters(queryParams map[string][]string) map[string]interface{} {
	filters := make(map[string]interface{})
	
	// Filter by name
	if names, ok := queryParams["name"]; ok && len(names) > 0 {
		filters["name"] = names[0]
	}
	
	// Filter by repository URL
	if repoURLs, ok := queryParams["repository_url"]; ok && len(repoURLs) > 0 {
		filters["repository.url"] = repoURLs[0]
	}
	
	// Filter by repository source
	if repoSources, ok := queryParams["repository_source"]; ok && len(repoSources) > 0 {
		filters["repository.source"] = repoSources[0]
	}
	
	// Filter by version
	if versions, ok := queryParams["version"]; ok && len(versions) > 0 {
		filters["version"] = versions[0]
	}
	
	// Filter by latest only
	if latests, ok := queryParams["latest"]; ok && len(latests) > 0 {
		if latests[0] == "true" {
			filters["version_detail.is_latest"] = true
		} else if latests[0] == "false" {
			filters["version_detail.is_latest"] = false
		}
	}
	
	// Filter by package registry (npm, docker, pypi, etc.)
	if packageRegistries, ok := queryParams["package_registry"]; ok && len(packageRegistries) > 0 {
		filters["package_registry"] = packageRegistries[0]
	}
	
	return filters
}