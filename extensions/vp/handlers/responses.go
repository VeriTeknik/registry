package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
)

// StandardResponse represents a standard API response
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// FeedbackListResponse represents the response for feedback listing
type FeedbackListResponse struct {
	Feedback   []stats.ServerFeedback `json:"feedback"`
	TotalCount int                    `json:"total_count"`
	HasMore    bool                   `json:"has_more"`
}

// UserFeedbackResponse represents the response for user feedback check
type UserFeedbackResponse struct {
	HasRated bool                   `json:"has_rated"`
	Feedback *stats.ServerFeedback  `json:"feedback,omitempty"`
}

// FeedbackSubmissionResponse represents the response for feedback submission
type FeedbackSubmissionResponse struct {
	Success  bool                   `json:"success"`
	Message  string                 `json:"message"`
	Feedback *stats.ServerFeedback  `json:"feedback,omitempty"`
	Stats    *stats.ServerStats     `json:"stats,omitempty"`
}

// InstallTrackingResponse represents the response for installation tracking
type InstallTrackingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StatsResponse represents the response for server stats
type StatsResponse struct {
	Stats *stats.ServerStats `json:"stats"`
}

// Helper functions for common response patterns

// WriteStandardResponse writes a standard JSON response
func WriteStandardResponse(w http.ResponseWriter, success bool, message string, data interface{}) error {
	response := StandardResponse{
		Success: success,
		Message: message,
		Data:    data,
	}
	return writeJSONResponse(w, response)
}

// WriteStandardError writes a standard error response
func WriteStandardError(w http.ResponseWriter, status int, message string) {
	response := StandardResponse{
		Success: false,
		Error:   message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// WriteFeedbackListResponse writes a feedback list response
func WriteFeedbackListResponse(w http.ResponseWriter, feedback []stats.ServerFeedback, totalCount int, hasMore bool) error {
	response := FeedbackListResponse{
		Feedback:   feedback,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}
	return writeJSONResponse(w, response)
}

// WriteUserFeedbackResponse writes a user feedback check response
func WriteUserFeedbackResponse(w http.ResponseWriter, hasRated bool, feedback *stats.ServerFeedback) error {
	response := UserFeedbackResponse{
		HasRated: hasRated,
		Feedback: feedback,
	}
	return writeJSONResponse(w, response)
}

// WriteFeedbackSubmissionResponse writes a feedback submission response
func WriteFeedbackSubmissionResponse(w http.ResponseWriter, success bool, message string, feedback *stats.ServerFeedback, stats *stats.ServerStats) error {
	response := FeedbackSubmissionResponse{
		Success:  success,
		Message:  message,
		Feedback: feedback,
		Stats:    stats,
	}
	return writeJSONResponse(w, response)
}

// WriteInstallTrackingResponse writes an installation tracking response
func WriteInstallTrackingResponse(w http.ResponseWriter, success bool, message string) error {
	response := InstallTrackingResponse{
		Success: success,
		Message: message,
	}
	return writeJSONResponse(w, response)
}

// WriteStatsResponse writes a stats response
func WriteStatsResponse(w http.ResponseWriter, stats *stats.ServerStats) error {
	response := StatsResponse{
		Stats: stats,
	}
	return writeJSONResponse(w, response)
}

// WriteExtendedServersResponse writes a servers list response with stats
func WriteExtendedServersResponse(w http.ResponseWriter, servers []vpmodel.ExtendedServer) error {
	response := vpmodel.ExtendedServersResponse{
		Servers: servers,
	}
	return writeJSONResponse(w, response)
}

// WriteExtendedServerResponse writes a single server response with stats
func WriteExtendedServerResponse(w http.ResponseWriter, server vpmodel.ExtendedServer) error {
	response := vpmodel.ExtendedServerResponse{
		Server: server,
	}
	return writeJSONResponse(w, response)
}

// WriteCachedResponse writes a response with cache headers
func WriteCachedResponse(w http.ResponseWriter, data interface{}, cacheHit bool) error {
	if cacheHit {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	return writeJSONResponse(w, data)
}