package stats

import (
	"time"
)

// Source types for server stats
const (
	SourceRegistry  = "REGISTRY"
	SourceCommunity = "COMMUNITY"
)

// ServerStats represents statistics for a single server
type ServerStats struct {
	ServerID          string    `json:"server_id" bson:"server_id"`
	Source            string    `json:"source" bson:"source"` // REGISTRY or COMMUNITY
	InstallationCount int       `json:"installation_count" bson:"installation_count"`
	Rating            float64   `json:"rating" bson:"rating"`
	RatingCount       int       `json:"rating_count" bson:"rating_count"`
	FirstSeen         time.Time `json:"first_seen" bson:"first_seen"`
	LastUpdated       time.Time `json:"last_updated" bson:"last_updated"`

	// Analytics-derived metrics (synced from analytics)
	ActiveInstalls     int `json:"active_installs,omitempty" bson:"active_installs,omitempty"`
	DailyActiveUsers   int `json:"daily_active_users,omitempty" bson:"daily_active_users,omitempty"`
	MonthlyActiveUsers int `json:"monthly_active_users,omitempty" bson:"monthly_active_users,omitempty"`

	// For claimed servers, track the original source
	ClaimedFrom string    `json:"claimed_from,omitempty" bson:"claimed_from,omitempty"`
	ClaimedAt   time.Time `json:"claimed_at,omitempty" bson:"claimed_at,omitempty"`
}

// RatingRequest represents a request to submit a rating with optional comment
type RatingRequest struct {
	Rating    float64 `json:"rating" validate:"required,min=1,max=5"`
	Comment   string  `json:"comment,omitempty" validate:"max=1000"`  // Optional user comment
	Source    string  `json:"source,omitempty"`                      // REGISTRY or COMMUNITY, defaults to REGISTRY
	UserID    string  `json:"user_id,omitempty"`                     // User identifier for tracking
	Timestamp string  `json:"timestamp,omitempty"`                   // Client-provided timestamp
}

// InstallRequest represents an installation tracking request
type InstallRequest struct {
	Source    string `json:"source,omitempty"` // REGISTRY or COMMUNITY, defaults to REGISTRY
	UserID    string `json:"user_id,omitempty"`
	Version   string `json:"version,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// StatsResponse wraps ServerStats for API responses
type StatsResponse struct {
	Stats *ServerStats `json:"stats"`
}

// ClaimRequest represents a request to claim a community server
type ClaimRequest struct {
	PublishRequest  interface{} `json:"publish_request"`
	TransferStats   bool        `json:"transfer_stats"`
	CommunityStats  *ServerStats `json:"community_stats,omitempty"`
}

// ClaimResponse represents the response after claiming a server
type ClaimResponse struct {
	Success           bool         `json:"success"`
	ServerID          string       `json:"server_id"`
	TransferredStats  *ServerStats `json:"transferred_stats,omitempty"`
}

// GlobalStats represents aggregate statistics for the entire registry
type GlobalStats struct {
	TotalServers       int       `json:"total_servers"`
	TotalInstalls      int       `json:"total_installs"`
	ActiveServers      int       `json:"active_servers"`
	AverageRating      float64   `json:"average_rating"`
	LastUpdated        time.Time `json:"last_updated"`
}


// LeaderboardEntry represents an entry in various leaderboards
type LeaderboardEntry struct {
	ServerID    string      `json:"server_id"`
	MetricValue interface{} `json:"metric_value"`
	Rank        int         `json:"rank"`
}

// LeaderboardType defines the types of leaderboards available
type LeaderboardType string

const (
	LeaderboardTypeInstalls LeaderboardType = "installs"
	LeaderboardTypeRating   LeaderboardType = "rating"
	LeaderboardTypeActive   LeaderboardType = "active"
	LeaderboardTypeTrending LeaderboardType = "trending"
)

// StatsUpdateRequest represents a request to update server stats
type StatsUpdateRequest struct {
	InstallationDelta  int       `json:"installation_delta,omitempty"`
	ActiveInstalls     int       `json:"active_installs,omitempty"`
	DailyActiveUsers   int       `json:"daily_active_users,omitempty"`
	MonthlyActiveUsers int       `json:"monthly_active_users,omitempty"`
	LastUpdated        time.Time `json:"last_updated"`
}

// BatchStatsResponse represents multiple server stats
type BatchStatsResponse struct {
	Stats map[string]*ServerStats `json:"stats"`
}

// StatsTransferRequest represents a request to transfer stats during claiming
type StatsTransferRequest struct {
	FromServerID string `json:"from_server_id"`
	ToServerID   string `json:"to_server_id"`
	FromSource   string `json:"from_source"`
	ToSource     string `json:"to_source"`
}

// AggregatedStats represents combined stats from multiple sources
type AggregatedStats struct {
	ServerID          string              `json:"server_id"`
	TotalInstalls     int                 `json:"total_installs"`
	AverageRating     float64             `json:"average_rating"`
	TotalRatingCount  int                 `json:"total_rating_count"`
	SourceBreakdown   map[string]*ServerStats `json:"source_breakdown"`
	LastUpdated       time.Time           `json:"last_updated"`
}