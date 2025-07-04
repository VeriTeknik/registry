package stats

import (
	"time"
)

// ServerStats represents statistics for a single server
type ServerStats struct {
	ServerID          string    `json:"server_id" bson:"server_id"`
	InstallationCount int       `json:"installation_count" bson:"installation_count"`
	Rating            float64   `json:"rating" bson:"rating"`
	RatingCount       int       `json:"rating_count" bson:"rating_count"`
	LastUpdated       time.Time `json:"last_updated" bson:"last_updated"`

	// Analytics-derived metrics (synced from analytics)
	ActiveInstalls     int `json:"active_installs,omitempty" bson:"active_installs,omitempty"`
	DailyActiveUsers   int `json:"daily_active_users,omitempty" bson:"daily_active_users,omitempty"`
	MonthlyActiveUsers int `json:"monthly_active_users,omitempty" bson:"monthly_active_users,omitempty"`
}

// RatingRequest represents a request to submit a rating
type RatingRequest struct {
	Rating float64 `json:"rating" validate:"required,min=1,max=5"`
}

// InstallRequest represents an installation tracking request
type InstallRequest struct {
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

// TrendingServer represents a server with trending metrics
type TrendingServer struct {
	ServerID     string  `json:"server_id"`
	TrendScore   float64 `json:"trend_score"`
	WeeklyGrowth float64 `json:"weekly_growth"`
	InstallDelta int     `json:"install_delta"`
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