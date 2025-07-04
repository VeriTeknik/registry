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
	"github.com/modelcontextprotocol/registry/extensions/vp/model"
	"github.com/modelcontextprotocol/registry/internal/auth"
	"github.com/modelcontextprotocol/registry/internal/service"
	"github.com/modelcontextprotocol/registry/internal/types"
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

	// Verify GitHub token and get user info
	githubUser, err := auth.VerifyGitHubToken(token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Parse claim request
	var claimReq model.ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&claimReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate publish request
	if err := validatePublishRequest(&claimReq.PublishRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid publish request: %v", err), http.StatusBadRequest)
		return
	}

	// Get the community server being claimed
	communityServer, err := h.service.GetServerByID(r.Context(), serverID)
	if err != nil {
		if err == service.ErrServerNotFound {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get server: %v", err), http.StatusInternalServerError)
		return
	}

	// Verify this is a community server (no author)
	if communityServer.Author != "" {
		http.Error(w, "This server is already claimed", http.StatusBadRequest)
		return
	}

	// Verify the claiming user has access to the repository
	if !verifyRepositoryAccess(githubUser.Login, claimReq.PublishRequest.Repository) {
		http.Error(w, "You don't have access to the specified repository", http.StatusForbidden)
		return
	}

	// Create new server entry with author
	newServer := types.Server{
		ID:          serverID, // Keep the same ID
		Name:        claimReq.PublishRequest.Name,
		Description: claimReq.PublishRequest.Description,
		Repository:  claimReq.PublishRequest.Repository,
		Version:     claimReq.PublishRequest.Version,
		VersionDetail: types.VersionDetail{
			SchemaVersion: claimReq.PublishRequest.SchemaVersion,
			InstallType:   claimReq.PublishRequest.InstallType,
			InstallUrl:    claimReq.PublishRequest.InstallUrl,
			Transport:     claimReq.PublishRequest.Transport,
			IconUrl:       claimReq.PublishRequest.IconUrl,
		},
		Author:    githubUser.Login,
		CreatedAt: communityServer.CreatedAt, // Preserve original creation date
		UpdatedAt: time.Now(),
	}

	// Update the server in the database
	if err := h.service.UpdateServer(r.Context(), &newServer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to claim server: %v", err), http.StatusInternalServerError)
		return
	}

	// Handle stats transfer if requested
	var transferredStats *stats.ServerStats
	if claimReq.TransferStats {
		// Get current stats
		currentStats, err := h.statsDB.GetStats(r.Context(), serverID)
		if err == nil && (currentStats.InstallationCount > 0 || currentStats.RatingCount > 0) {
			transferredStats = currentStats
			// Stats are preserved since we're keeping the same server ID
		}
	}

	// Invalidate cache
	h.statsCache.Delete(fmt.Sprintf("vp:server:%s", serverID))
	h.statsCache.Delete("vp:servers:")

	// Get updated server with stats
	serverStats, _ := h.statsDB.GetStats(r.Context(), serverID)
	extendedServer := model.NewExtendedServer(&newServer, serverStats)

	// Return success response
	response := model.ClaimResponse{
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
	var verifyReq model.ClaimVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&verifyReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify GitHub token
	githubUser, err := auth.VerifyGitHubToken(verifyReq.GitHubToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

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

	response := model.ClaimVerificationResponse{
		VerificationCode: verificationCode,
		Instructions: fmt.Sprintf(
			"Add a file named '.mcp-claim-verification' to the root of your repository with the following content:\n%s\n\nThis code expires in 15 minutes.",
			verificationCode,
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
func validatePublishRequest(req *types.PublishRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if req.Repository.Owner == "" || req.Repository.Name == "" {
		return fmt.Errorf("repository owner and name are required")
	}
	if req.SchemaVersion == "" {
		return fmt.Errorf("schema version is required")
	}
	return nil
}

// verifyRepositoryAccess checks if the user has access to the repository
func verifyRepositoryAccess(username string, repo types.Repository) bool {
	// In production, this would make a GitHub API call to verify access
	// For now, we'll do a simple check
	return strings.EqualFold(repo.Owner, username)
}