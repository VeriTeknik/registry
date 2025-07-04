// Package validation provides utilities for validating and sanitizing database inputs
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// AllowedSourceValues defines the only allowed values for source parameters
var AllowedSourceValues = map[string]bool{
	"REGISTRY":  true,
	"COMMUNITY": true,
	"ALL":       true,
	"":          true, // Empty is allowed and defaults to REGISTRY
}

// ValidateSource validates and returns a safe source value for database queries
func ValidateSource(source string) (string, error) {
	source = strings.TrimSpace(source)
	
	// Check against whitelist
	if !AllowedSourceValues[source] {
		return "", fmt.Errorf("invalid source value: %s", source)
	}
	
	// Additional safety check - ensure no special characters
	if !isAlphanumeric(source) && source != "" {
		return "", fmt.Errorf("source contains invalid characters")
	}
	
	return source, nil
}

// SanitizeID validates and sanitizes an ID parameter for database queries
func SanitizeID(id string) (string, error) {
	id = strings.TrimSpace(id)
	
	if id == "" {
		return "", fmt.Errorf("ID cannot be empty")
	}
	
	// UUID pattern validation (with hyphens)
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if uuidPattern.MatchString(id) {
		return strings.ToLower(id), nil
	}
	
	// Alternative ID format (alphanumeric with dots, hyphens, underscores)
	idPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,254}$`)
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("ID contains invalid characters or format")
	}
	
	return id, nil
}

// SanitizeServerID specifically validates server IDs
func SanitizeServerID(serverID string) (string, error) {
	return SanitizeID(serverID)
}

// ValidateLimit ensures limit values are within acceptable bounds
func ValidateLimit(limit int) (int, error) {
	if limit < 1 {
		return 0, fmt.Errorf("limit must be at least 1")
	}
	if limit > 1000 {
		return 0, fmt.Errorf("limit cannot exceed 1000")
	}
	return limit, nil
}

// isAlphanumeric checks if a string contains only letters
func isAlphanumeric(s string) bool {
	if s == "" {
		return true
	}
	alphaPattern := regexp.MustCompile(`^[A-Z]+$`)
	return alphaPattern.MatchString(s)
}

// CreateSafeFilter creates a safe MongoDB filter for source queries
func CreateSafeFilter(source string) (map[string]interface{}, error) {
	validatedSource, err := ValidateSource(source)
	if err != nil {
		return nil, err
	}
	
	filter := make(map[string]interface{})
	if validatedSource != "" && validatedSource != "ALL" {
		filter["source"] = validatedSource
	}
	
	return filter, nil
}