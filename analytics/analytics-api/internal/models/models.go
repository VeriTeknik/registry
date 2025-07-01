package models

import (
	"time"
)

// TrackEventRequest represents an event tracking request
type TrackEventRequest struct {
	EventType  string                 `json:"event_type" binding:"required,oneof=install uninstall usage error view"`
	ServerID   string                 `json:"server_id" binding:"required"`
	ClientID   string                 `json:"client_id" binding:"required"`
	SessionID  string                 `json:"session_id"`
	UserID     string                 `json:"user_id"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Event represents an analytics event
type Event struct {
	Timestamp  time.Time              `json:"timestamp"`
	EventType  string                 `json:"event_type"`
	ServerID   string                 `json:"server_id"`
	ServerName string                 `json:"server_name"`
	ClientID   string                 `json:"client_id"`
	SessionID  string                 `json:"session_id"`
	UserID     string                 `json:"user_id"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ServerStats represents aggregated server statistics
type ServerStats struct {
	ServerID        string    `json:"server_id"`
	ServerName      string    `json:"server_name"`
	TotalInstalls   int64     `json:"total_installs"`
	ActiveInstalls  int64     `json:"active_installs"`
	TotalUsage      int64     `json:"total_usage"`
	DailyActiveUsers int64    `json:"daily_active_users"`
	MonthlyActiveUsers int64  `json:"monthly_active_users"`
	AverageRating   float64   `json:"average_rating"`
	RatingCount     int64     `json:"rating_count"`
	CommentCount    int64     `json:"comment_count"`
	LastUpdated     time.Time `json:"last_updated"`
}

// TimelineData represents time-series data
type TimelineData struct {
	Date     string `json:"date"`
	Installs int64  `json:"installs"`
	Usage    int64  `json:"usage"`
	Errors   int64  `json:"errors"`
	Users    int64  `json:"users"`
}

// TrendingServer represents a trending server
type TrendingServer struct {
	ServerID       string  `json:"server_id"`
	ServerName     string  `json:"server_name"`
	Description    string  `json:"description"`
	TrendingScore  float64 `json:"trending_score"`
	InstallGrowth  float64 `json:"install_growth"`
	UsageGrowth    float64 `json:"usage_growth"`
	RecentInstalls int64   `json:"recent_installs"`
}

// Rating represents a user rating
type Rating struct {
	ServerID  string    `json:"server_id"`
	UserID    string    `json:"user_id"`
	Rating    int       `json:"rating" binding:"min=1,max=5"`
	Comment   string    `json:"comment"`
	Timestamp time.Time `json:"timestamp"`
}

// Comment represents a user comment
type Comment struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	UserID      string    `json:"user_id"`
	Comment     string    `json:"comment" binding:"required,min=1,max=1000"`
	ParentID    string    `json:"parent_id"`
	Timestamp   time.Time `json:"timestamp"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsVerified  bool      `json:"is_verified"`
	HelpfulCount int      `json:"helpful_count"`
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query          string   `form:"q"`
	Categories     []string `form:"categories"`
	PackageTypes   []string `form:"package_types"`
	MinRating      float64  `form:"min_rating"`
	SortBy         string   `form:"sort_by"`
	Offset         int      `form:"offset"`
	Limit          int      `form:"limit"`
}

// SearchResult represents search results
type SearchResult struct {
	Servers    []ServerSearchResult `json:"servers"`
	TotalCount int64               `json:"total_count"`
	Took       int64               `json:"took_ms"`
}