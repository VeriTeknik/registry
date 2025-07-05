package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/registry/extensions/stats"
)

// SubmitFeedbackHandler handles rating submission with optional comments
func (h *VPHandlers) SubmitFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if feedback database is initialized
	if h.feedbackDB == nil {
		fmt.Printf("Warning: Feedback database not initialized in SubmitFeedbackHandler\n")
		// Fall back to basic rating without feedback tracking
		h.handleBasicRating(w, r)
		return
	}

	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Parse rating request
	var ratingReq stats.RatingRequest
	if err := json.NewDecoder(r.Body).Decode(&ratingReq); err != nil {
		fmt.Printf("Error decoding rating request: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Debug log
	fmt.Printf("Received rating request: %+v\n", ratingReq)

	// Validate rating
	if ratingReq.Rating < 1 || ratingReq.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Validate comment length
	if len(ratingReq.Comment) > 1000 {
		http.Error(w, "Comment must not exceed 1000 characters", http.StatusBadRequest)
		return
	}

	// Validate source if provided
	if ratingReq.Source != "" && ratingReq.Source != stats.SourceRegistry && ratingReq.Source != stats.SourceCommunity {
		http.Error(w, "Invalid source. Must be 'REGISTRY' or 'COMMUNITY'", http.StatusBadRequest)
		return
	}

	// Default source to REGISTRY if not specified
	if ratingReq.Source == "" {
		ratingReq.Source = stats.SourceRegistry
	}

	// Validate user ID
	if ratingReq.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Check for existing feedback from this user
	existingFeedback, err := h.feedbackDB.GetUserFeedback(r.Context(), serverID, ratingReq.UserID, ratingReq.Source)
	if err != nil && err != stats.ErrFeedbackNotFound {
		http.Error(w, "Failed to check existing feedback", http.StatusInternalServerError)
		return
	}

	// Create or update feedback
	feedback := &stats.ServerFeedback{
		ServerID:  serverID,
		Source:    ratingReq.Source,
		UserID:    ratingReq.UserID,
		Rating:    ratingReq.Rating,
		Comment:   ratingReq.Comment,
		IsPublic:  true,
	}

	if existingFeedback != nil {
		// Update existing feedback
		feedback.ID = existingFeedback.ID
		feedback.CreatedAt = existingFeedback.CreatedAt
		err = h.feedbackDB.UpdateFeedback(r.Context(), feedback)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update feedback: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// Create new feedback
		feedback.ID = uuid.New().String()
		err = h.feedbackDB.CreateFeedback(r.Context(), feedback)
		if err != nil {
			if err == stats.ErrDuplicateFeedback {
				http.Error(w, "You have already rated this server", http.StatusConflict)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to create feedback: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update rating statistics with source
	if err := h.statsDB.UpdateRating(r.Context(), serverID, ratingReq.Source, ratingReq.Rating); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update stats: %v\n", err)
	}

	// Invalidate cache
	h.invalidateFeedbackCache(serverID, ratingReq.Source)

	// Get updated stats
	updatedStats, err := h.statsDB.GetStats(r.Context(), serverID, ratingReq.Source)
	if err != nil {
		// Still return success even if we can't get updated stats
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"message":  "Feedback submitted successfully",
			"feedback": feedback,
		})
		return
	}

	// Return success with feedback and updated stats
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Feedback submitted successfully",
		"feedback": feedback,
		"stats":    updatedStats,
	})
}

// GetServerFeedbackHandler retrieves all feedback for a server
func (h *VPHandlers) GetServerFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] != "feedback" {
		http.Error(w, "Invalid feedback path", http.StatusBadRequest)
		return
	}
	serverID := parts[0]
	
	// Log for debugging
	fmt.Printf("GetServerFeedbackHandler called - ServerID: %s, Path: %s\n", serverID, path)
	
	// Check if feedback database is initialized
	if h.feedbackDB == nil {
		fmt.Printf("Error: Feedback database is nil in GetServerFeedbackHandler\n")
		http.Error(w, "Feedback service unavailable", http.StatusInternalServerError)
		return
	}

	// Get query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	sortStr := r.URL.Query().Get("sort")
	sort := stats.FeedbackSortNewest
	switch sortStr {
	case "oldest":
		sort = stats.FeedbackSortOldest
	case "rating_high":
		sort = stats.FeedbackSortRatingHigh
	case "rating_low":
		sort = stats.FeedbackSortRatingLow
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = stats.SourceRegistry
	}

	// Build cache key
	cacheKey := fmt.Sprintf("vp:feedback:%s:%s:%d:%d:%s", serverID, source, limit, offset, sort)
	if cached, found := h.statsCache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get feedback from database
	feedbackResponse, err := h.feedbackDB.GetServerFeedback(r.Context(), serverID, source, limit, offset, sort)
	if err != nil {
		fmt.Printf("Error getting feedback for server %s: %v\n", serverID, err)
		// Return empty response on error
		feedbackResponse = &stats.FeedbackResponse{
			Feedback:   []*stats.ServerFeedback{},
			TotalCount: 0,
			HasMore:    false,
		}
	}

	// Ensure feedback array is not nil
	if feedbackResponse.Feedback == nil {
		feedbackResponse.Feedback = []*stats.ServerFeedback{}
	}
	
	// Cache the response
	h.statsCache.Set(cacheKey, feedbackResponse)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(feedbackResponse); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetUserFeedbackHandler checks if a user has rated a server
func (h *VPHandlers) GetUserFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	// Extract server ID and user ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] == "" {
		http.Error(w, "Server ID and User ID are required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]
	userID := parts[2]

	source := r.URL.Query().Get("source")
	if source == "" {
		source = stats.SourceRegistry
	}

	// Check for user's feedback
	feedback, err := h.feedbackDB.GetUserFeedback(r.Context(), serverID, userID, source)
	if err != nil {
		if err == stats.ErrFeedbackNotFound {
			// User hasn't rated yet
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats.UserFeedbackResponse{
				HasRated: false,
			})
			return
		}
		http.Error(w, fmt.Sprintf("Failed to check user feedback: %v", err), http.StatusInternalServerError)
		return
	}

	// Return user's feedback
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats.UserFeedbackResponse{
		HasRated: true,
		Feedback: feedback,
	})
}

// UpdateFeedbackHandler allows users to update their feedback
func (h *VPHandlers) UpdateFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID and feedback ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] == "" {
		http.Error(w, "Server ID and Feedback ID are required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]
	feedbackID := parts[2]

	// Parse update request
	var updateReq stats.FeedbackUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate rating
	if updateReq.Rating < 1 || updateReq.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Validate comment length
	if len(updateReq.Comment) > 1000 {
		http.Error(w, "Comment must not exceed 1000 characters", http.StatusBadRequest)
		return
	}

	// Validate user ID
	if updateReq.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Get existing feedback to verify ownership
	existingFeedback, err := h.feedbackDB.GetFeedback(r.Context(), feedbackID)
	if err != nil {
		if err == stats.ErrFeedbackNotFound {
			http.Error(w, "Feedback not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get feedback: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify user owns this feedback
	if existingFeedback.UserID != updateReq.UserID {
		http.Error(w, "Unauthorized to update this feedback", http.StatusForbidden)
		return
	}

	// Update feedback
	existingFeedback.Rating = updateReq.Rating
	existingFeedback.Comment = updateReq.Comment

	if err := h.feedbackDB.UpdateFeedback(r.Context(), existingFeedback); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update feedback: %v", err), http.StatusInternalServerError)
		return
	}

	// Update rating statistics
	if err := h.statsDB.UpdateRating(r.Context(), serverID, existingFeedback.Source, updateReq.Rating); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update stats: %v\n", err)
	}

	// Invalidate cache
	h.invalidateFeedbackCache(serverID, existingFeedback.Source)

	// Return updated feedback
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Feedback updated successfully",
		"feedback": existingFeedback,
	})
}

// DeleteFeedbackHandler allows users to delete their feedback
func (h *VPHandlers) DeleteFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract server ID and feedback ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] == "" {
		http.Error(w, "Server ID and Feedback ID are required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]
	feedbackID := parts[2]

	// Get user ID from query params or request body
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Delete feedback
	if err := h.feedbackDB.DeleteFeedback(r.Context(), feedbackID, userID); err != nil {
		if err == stats.ErrFeedbackNotFound {
			http.Error(w, "Feedback not found or unauthorized", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete feedback: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO: Recalculate rating statistics after deletion

	// Invalidate cache
	h.invalidateFeedbackCache(serverID, "")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Feedback deleted successfully",
	})
}

// invalidateFeedbackCache invalidates feedback-related cache entries
func (h *VPHandlers) invalidateFeedbackCache(serverID string, source string) {
	// Invalidate feedback cache patterns
	h.statsCache.Delete(fmt.Sprintf("vp:feedback:%s:*", serverID))
	
	// Also invalidate stats cache
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s", serverID))
	if source != "" {
		h.statsCache.Delete(fmt.Sprintf("vp:stats:%s:%s", serverID, source))
	}
	h.statsCache.Delete("vp:servers:") // Clear servers list cache
}

// handleBasicRating handles simple rating without feedback tracking
func (h *VPHandlers) handleBasicRating(w http.ResponseWriter, r *http.Request) {
	// Extract server ID from URL
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/vp/servers/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		http.Error(w, "Server ID is required", http.StatusBadRequest)
		return
	}
	serverID := parts[0]

	// Parse rating request
	var ratingReq stats.RatingRequest
	if err := json.NewDecoder(r.Body).Decode(&ratingReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate rating
	if ratingReq.Rating < 1 || ratingReq.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Default source to REGISTRY if not specified
	if ratingReq.Source == "" {
		ratingReq.Source = stats.SourceRegistry
	}

	// Update rating statistics
	if err := h.statsDB.UpdateRating(r.Context(), serverID, ratingReq.Source, ratingReq.Rating); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update rating: %v", err), http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	h.invalidateFeedbackCache(serverID, ratingReq.Source)

	// Get updated stats
	updatedStats, err := h.statsDB.GetStats(r.Context(), serverID, ratingReq.Source)
	if err != nil {
		// Still return success even if we can't get updated stats
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Rating submitted successfully",
		})
		return
	}

	// Return success with updated stats
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rating submitted successfully",
		"stats":   updatedStats,
	})
}