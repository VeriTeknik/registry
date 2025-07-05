package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/registry/extensions/stats"
)

// Common HTTP method validation
func validateHTTPMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// Extract server ID from VP server paths like "/vp/servers/{id}/..."
func extractServerIDFromPath(path string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return "", fmt.Errorf("server ID is required")
	}
	return parts[0], nil
}

// Extract path segments from VP server paths for routing
func extractPathSegments(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
}

// Validate source parameter (REGISTRY, COMMUNITY, or empty)
func validateSource(source string) error {
	if source != "" && source != stats.SourceRegistry && source != stats.SourceCommunity {
		return fmt.Errorf("invalid source. Must be 'REGISTRY' or 'COMMUNITY'")
	}
	return nil
}

// Parse and validate limit parameter with default and max values
func parseLimit(limitStr string, defaultLimit, maxLimit int) int {
	if limitStr == "" {
		return defaultLimit
	}
	
	if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= maxLimit {
		return parsedLimit
	}
	
	return defaultLimit
}

// Parse offset parameter for pagination
func parseOffset(offsetStr string) int {
	if offsetStr == "" {
		return 0
	}
	
	if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
		return parsedOffset
	}
	
	return 0
}

// Standard JSON response helpers
func writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func writeJSONResponseWithStatus(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// Standard success response
func writeSuccessResponse(w http.ResponseWriter, message string) error {
	return writeJSONResponse(w, map[string]interface{}{
		"success": true,
		"message": message,
	})
}

// Standard error response with JSON
func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// Validate rating value (1-5)
func validateRating(rating float64) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	return nil
}

// Validate comment length
func validateComment(comment string, maxLength int) error {
	if len(comment) > maxLength {
		return fmt.Errorf("comment exceeds maximum length of %d characters", maxLength)
	}
	return nil
}

// Common cache key generators
func generateServerCacheKey(serverID string) string {
	return fmt.Sprintf("vp:server:%s", serverID)
}

func generateStatsCacheKey(serverID string) string {
	return fmt.Sprintf("vp:stats:%s", serverID)
}

func generateSourceStatsCacheKey(serverID, source string) string {
	return fmt.Sprintf("vp:stats:%s:%s", serverID, source)
}

func generateFeedbackCacheKey(serverID, source string) string {
	return fmt.Sprintf("vp:feedback:%s:%s", serverID, source)
}

// Common cache invalidation patterns
type CacheInvalidator interface {
	Delete(key string)
}

func invalidateServerCaches(cache CacheInvalidator, serverID string) {
	cache.Delete(generateServerCacheKey(serverID))
	cache.Delete(generateStatsCacheKey(serverID))
	cache.Delete(fmt.Sprintf("vp:stats:%s:aggregated", serverID))
	cache.Delete("vp:servers:")
	cache.Delete("vp:stats:global")
}

func invalidateSourceCaches(cache CacheInvalidator, serverID, source string) {
	cache.Delete(generateSourceStatsCacheKey(serverID, source))
	cache.Delete(fmt.Sprintf("vp:stats:global:%s", source))
	cache.Delete(generateFeedbackCacheKey(serverID, source))
}