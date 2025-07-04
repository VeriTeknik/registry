package model

import (
	"github.com/modelcontextprotocol/registry/internal/model"
	"github.com/modelcontextprotocol/registry/extensions/stats"
)

// ClaimRequest represents a request to claim a community server
type ClaimRequest struct {
	// PublishRequest contains the standard publish request fields
	PublishRequest model.PublishRequest `json:"publish_request" validate:"required"`
	
	// TransferStats indicates whether to transfer stats from the community server
	TransferStats bool `json:"transfer_stats"`
	
	// VerificationCode is optional, used to verify ownership of the repository
	VerificationCode string `json:"verification_code,omitempty"`
}

// ClaimResponse represents the response after claiming a server
type ClaimResponse struct {
	Success          bool                `json:"success"`
	ServerID         string              `json:"server_id"`
	Message          string              `json:"message,omitempty"`
	TransferredStats *stats.ServerStats  `json:"transferred_stats,omitempty"`
	NewServer        *ExtendedServer     `json:"new_server,omitempty"`
}

// ClaimVerificationRequest represents a request to verify ownership for claiming
type ClaimVerificationRequest struct {
	ServerID     string `json:"server_id" validate:"required"`
	GitHubToken  string `json:"github_token" validate:"required"`
}

// ClaimVerificationResponse contains the verification code to be added to the repository
type ClaimVerificationResponse struct {
	VerificationCode string `json:"verification_code"`
	Instructions     string `json:"instructions"`
	ExpiresAt        int64  `json:"expires_at"`
}