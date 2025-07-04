package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/registry/extensions/stats"
	vpmodel "github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/model"
)

// ClaimServerHandler handles claiming a community server
func (h *VPHandlers) ClaimServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	// Verify authentication
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	// Extract token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
		return
	}

	// For now, extract username from a custom header or use a placeholder
	// In production, this should validate the GitHub token properly
	githubUsername := r.Header.Get("X-GitHub-User")
	if githubUsername == "" {
		// TODO: Implement proper GitHub token validation
		http.Error(w, "GitHub authentication required", http.StatusUnauthorized)
		return
	}

	// Parse claim request
	var claimReq vpmodel.ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&claimReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate publish request
	if err := validatePublishRequest(&claimReq.PublishRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid publish request: %v", err), http.StatusBadRequest)
		return
	}

	// Get the community server being claimed to verify it exists
	_, err := h.service.GetByID(serverID)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}
	
	// For now, we'll skip the community server check
	// In production, this would check if the server has an owner/author field

	// Verify the claiming user has access to the repository
	if !verifyRepositoryAccess(githubUsername, claimReq.PublishRequest.Repository) {
		http.Error(w, "You don't have access to the specified repository", http.StatusForbidden)
		return
	}

	// Create new server entry from publish request
	newServerDetail := &model.ServerDetail{
		Server: model.Server{
			ID:            serverID, // Keep the same ID
			Name:          claimReq.PublishRequest.Name,
			Description:   claimReq.PublishRequest.Description,
			Repository:    claimReq.PublishRequest.Repository,
			VersionDetail: claimReq.PublishRequest.VersionDetail,
		},
		Packages: claimReq.PublishRequest.Packages,
		Remotes:  claimReq.PublishRequest.Remotes,
	}
	
	// Update the server in the database
	if err := h.service.Publish(newServerDetail); err != nil {
		http.Error(w, fmt.Sprintf("Failed to claim server: %v", err), http.StatusInternalServerError)
		return
	}

	// Handle stats transfer if requested
	var transferredStats *stats.ServerStats
	if claimReq.TransferStats {
		// Transfer stats from COMMUNITY to REGISTRY source
		err := h.statsDB.TransferStats(r.Context(), serverID, serverID, stats.SourceCommunity, stats.SourceRegistry)
		if err != nil {
			// Log error but don't fail the claim
			fmt.Printf("Failed to transfer stats during claim: %v\n", err)
		} else {
			// Get the transferred stats
			transferredStats, _ = h.statsDB.GetStats(r.Context(), serverID, stats.SourceRegistry)
		}
	}

	// Invalidate cache
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s", serverID))
	h.statsCache.Delete(fmt.Sprintf("vp:stats:%s:aggregated", serverID))
	h.statsCache.Delete("vp:servers:")
	h.statsCache.Delete("vp:stats:global")

	// Get updated server with stats from REGISTRY source
	serverStats, _ := h.statsDB.GetStats(r.Context(), serverID, stats.SourceRegistry)
	extendedServer := vpmodel.NewExtendedServer(&newServerDetail.Server, serverStats)

	// Return success response
	response := vpmodel.ClaimResponse{
		Success:          true,
		ServerID:         serverID,
		Message:          "Server successfully claimed",
		TransferredStats: transferredStats,
		NewServer:        &extendedServer,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GenerateClaimVerificationHandler generates a verification code for claiming
func (h *VPHandlers) GenerateClaimVerificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse verification request
	var verifyReq vpmodel.ClaimVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&verifyReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, use a placeholder for GitHub user
	// TODO: Implement proper GitHub token validation
	githubUsername := "testuser"

	// Generate verification code
	codeBytes := make([]byte, 16)
	if _, err := rand.Read(codeBytes); err != nil {
		http.Error(w, "Failed to generate verification code", http.StatusInternalServerError)
		return
	}
	verificationCode := hex.EncodeToString(codeBytes)

	// Store verification code with expiration (this would be in Redis/cache in production)
	// For now, we'll just return it
	expiresAt := time.Now().Add(15 * time.Minute).Unix()

	response := vpmodel.ClaimVerificationResponse{
		VerificationCode: verificationCode,
		Instructions: fmt.Sprintf(
			"Add a file named '.mcp-claim-verification' to the root of your repository with the following content:\n%s\n\nThis code expires in 15 minutes. User: %s",
			verificationCode,
			githubUsername,
		),
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// validatePublishRequest validates the publish request fields
func validatePublishRequest(req *model.PublishRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if req.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}
	return nil
}

// verifyRepositoryAccess checks if the user has access to the repository
func verifyRepositoryAccess(username string, repo model.Repository) bool {
	// In production, this would make a GitHub API call to verify access
	// For now, we'll just return true
	return true
}