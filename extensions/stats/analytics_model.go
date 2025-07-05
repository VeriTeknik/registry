package stats

import (
	"time"
)

// AnalyticsMetrics represents comprehensive analytics data
type AnalyticsMetrics struct {
	// Core Metrics
	TotalInstalls     int64   `json:"total_installs" bson:"total_installs"`
	TotalAPICalls     int64   `json:"total_api_calls" bson:"total_api_calls"`
	ActiveUsers       int64   `json:"active_users" bson:"active_users"`
	ActiveInstalls    int64   `json:"active_installs" bson:"active_installs"`
	
	// Growth Metrics
	InstallsToday     int64   `json:"installs_today" bson:"installs_today"`
	InstallsThisWeek  int64   `json:"installs_this_week" bson:"installs_this_week"`
	InstallsThisMonth int64   `json:"installs_this_month" bson:"installs_this_month"`
	WeeklyGrowth      float64 `json:"weekly_growth" bson:"weekly_growth"`
	MonthlyGrowth     float64 `json:"monthly_growth" bson:"monthly_growth"`
	InstallVelocity   float64 `json:"install_velocity" bson:"install_velocity"` // Installs per hour
	
	// Quality Metrics
	AverageRating     float64 `json:"average_rating" bson:"average_rating"`
	TotalRatings      int64   `json:"total_ratings" bson:"total_ratings"`
	FiveStarServers   int64   `json:"five_star_servers" bson:"five_star_servers"`
	TotalFeedback     int64   `json:"total_feedback" bson:"total_feedback"`
	
	// Performance Metrics
	ResponseTimeP50   float64 `json:"response_time_p50" bson:"response_time_p50"`     // milliseconds
	ResponseTimeP90   float64 `json:"response_time_p90" bson:"response_time_p90"`     // milliseconds
	ResponseTimeP99   float64 `json:"response_time_p99" bson:"response_time_p99"`     // milliseconds
	UptimePercentage  float64 `json:"uptime_percentage" bson:"uptime_percentage"`
	ErrorRate         float64 `json:"error_rate" bson:"error_rate"`
	
	// Discovery Metrics
	TotalSearches     int64   `json:"total_searches" bson:"total_searches"`
	SearchSuccessRate float64 `json:"search_success_rate" bson:"search_success_rate"`
	RegistryInstalls  int64   `json:"registry_installs" bson:"registry_installs"`
	CommunityInstalls int64   `json:"community_installs" bson:"community_installs"`
	
	// Developer Activity
	ActivePublishers  int64   `json:"active_publishers" bson:"active_publishers"`
	NewServers        int64   `json:"new_servers" bson:"new_servers"`
	UpdatedServers    int64   `json:"updated_servers" bson:"updated_servers"`
	
	// Timestamp
	LastUpdated       time.Time `json:"last_updated" bson:"last_updated"`
}

// APICallMetrics tracks API endpoint usage
type APICallMetrics struct {
	Endpoint      string    `json:"endpoint" bson:"endpoint"`
	Method        string    `json:"method" bson:"method"`
	Count         int64     `json:"count" bson:"count"`
	AvgDuration   float64   `json:"avg_duration" bson:"avg_duration"` // milliseconds
	ErrorCount    int64     `json:"error_count" bson:"error_count"`
	LastCalled    time.Time `json:"last_called" bson:"last_called"`
}

// ActivityEvent represents a single activity in the system
type ActivityEvent struct {
	ID           string                 `json:"id" bson:"_id"`
	Type         string                 `json:"type" bson:"type"` // "install", "rating", "update", "search"
	ServerID     string                 `json:"server_id,omitempty" bson:"server_id,omitempty"`
	ServerName   string                 `json:"server_name,omitempty" bson:"server_name,omitempty"`
	UserID       string                 `json:"user_id,omitempty" bson:"user_id,omitempty"`
	Value        interface{}            `json:"value,omitempty" bson:"value,omitempty"` // rating value, search term, etc.
	Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp" bson:"timestamp"`
}

// CategoryStats represents statistics for a server category
type CategoryStats struct {
	Category          string    `json:"category" bson:"category"`
	ServerCount       int64     `json:"server_count" bson:"server_count"`
	TotalInstalls     int64     `json:"total_installs" bson:"total_installs"`
	AverageRating     float64   `json:"average_rating" bson:"average_rating"`
	WeeklyGrowth      float64   `json:"weekly_growth" bson:"weekly_growth"`
	LastUpdated       time.Time `json:"last_updated" bson:"last_updated"`
}

// ConversionFunnel tracks user journey metrics
type ConversionFunnel struct {
	Period            string    `json:"period" bson:"period"` // "day", "week", "month"
	Date              time.Time `json:"date" bson:"date"`
	TotalSearches     int64     `json:"total_searches" bson:"total_searches"`
	SearchToView      int64     `json:"search_to_view" bson:"search_to_view"`
	ViewToInstall     int64     `json:"view_to_install" bson:"view_to_install"`
	SearchToInstall   int64     `json:"search_to_install" bson:"search_to_install"`
	ConversionRate    float64   `json:"conversion_rate" bson:"conversion_rate"`
}

// TrendingServer represents a server with trending metrics
type TrendingServer struct {
	ServerID          string    `json:"server_id" bson:"server_id"`
	ServerName        string    `json:"server_name" bson:"server_name"`
	TrendingScore     float64   `json:"trending_score" bson:"trending_score"`
	TrendScore        float64   `json:"trend_score" bson:"trend_score"` // Alias for compatibility
	InstallsToday     int64     `json:"installs_today" bson:"installs_today"`
	InstallVelocity   float64   `json:"install_velocity" bson:"install_velocity"`
	MomentumChange    float64   `json:"momentum_change" bson:"momentum_change"` // % change in velocity
	RecentInstalls    int64     `json:"recent_installs" bson:"recent_installs"`
	PreviousInstalls  int64     `json:"previous_installs" bson:"previous_installs"`
	TrendPeriod       string    `json:"trend_period" bson:"trend_period"`
	Category          string    `json:"category,omitempty" bson:"category,omitempty"`
}

// SearchAnalytics tracks search behavior
type SearchAnalytics struct {
	SearchTerm        string    `json:"search_term" bson:"search_term"`
	Count             int64     `json:"count" bson:"count"`
	ResultsFound      int64     `json:"results_found" bson:"results_found"`
	InstallsFromSearch int64    `json:"installs_from_search" bson:"installs_from_search"`
	SuccessRate       float64   `json:"success_rate" bson:"success_rate"`
	LastSearched      time.Time `json:"last_searched" bson:"last_searched"`
}

// ServerHealthMetrics tracks individual server health
type ServerHealthMetrics struct {
	ServerID          string    `json:"server_id" bson:"server_id"`
	ResponseTime      float64   `json:"response_time" bson:"response_time"` // ms
	Availability      float64   `json:"availability" bson:"availability"`   // percentage
	ErrorRate         float64   `json:"error_rate" bson:"error_rate"`
	LastHealthCheck   time.Time `json:"last_health_check" bson:"last_health_check"`
	Status            string    `json:"status" bson:"status"` // "healthy", "degraded", "down"
}

// TimeSeriesData represents metrics over time
type TimeSeriesData struct {
	Timestamp         time.Time `json:"timestamp" bson:"timestamp"`
	Installs          int64     `json:"installs" bson:"installs"`
	APICalls          int64     `json:"api_calls" bson:"api_calls"`
	ActiveUsers       int64     `json:"active_users" bson:"active_users"`
	NewServers        int64     `json:"new_servers" bson:"new_servers"`
	Ratings           int64     `json:"ratings" bson:"ratings"`
}

// GrowthMetrics represents growth statistics
type GrowthMetrics struct {
	Metric              string      `json:"metric" bson:"metric"`  // "installs", "users", "api_calls", etc.
	Period              string      `json:"period" bson:"period"`  // "hour", "day", "week", "month"
	CurrentPeriodStart  time.Time   `json:"current_period_start" bson:"current_period_start"`
	PreviousPeriodStart time.Time   `json:"previous_period_start" bson:"previous_period_start"`
	CurrentValue        float64     `json:"current_value" bson:"current_value"`
	PreviousValue       float64     `json:"previous_value" bson:"previous_value"`
	AbsoluteChange      float64     `json:"absolute_change" bson:"absolute_change"`
	GrowthRate          float64     `json:"growth_rate" bson:"growth_rate"`  // Percentage
	Momentum            float64     `json:"momentum" bson:"momentum"`        // Acceleration/deceleration
	Trend               string      `json:"trend" bson:"trend"`              // "accelerating", "steady", "decelerating", "new"
	DataPoints          []DataPoint `json:"data_points" bson:"data_points"`  // For visualization
}

// MilestoneEvent represents significant achievements
type MilestoneEvent struct {
	ID                string    `json:"id" bson:"_id"`
	Type              string    `json:"type" bson:"type"` // "installs", "servers", "users", "ratings"
	Milestone         int64     `json:"milestone" bson:"milestone"` // 100, 1000, etc.
	AchievedAt        time.Time `json:"achieved_at" bson:"achieved_at"`
	ServerID          string    `json:"server_id,omitempty" bson:"server_id,omitempty"`
	ServerName        string    `json:"server_name,omitempty" bson:"server_name,omitempty"`
	Description       string    `json:"description" bson:"description"`
}

// AnalyticsResponse wraps analytics data for API responses
type AnalyticsResponse struct {
	Metrics           *AnalyticsMetrics      `json:"metrics"`
	TrendingServers   []TrendingServer       `json:"trending_servers,omitempty"`
	RecentActivity    []ActivityEvent        `json:"recent_activity,omitempty"`
	CategoryBreakdown []CategoryStats        `json:"category_breakdown,omitempty"`
	SearchInsights    []SearchAnalytics      `json:"search_insights,omitempty"`
	Milestones        []MilestoneEvent       `json:"milestones,omitempty"`
	TimePeriod        string                 `json:"time_period"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

// DashboardMetrics represents the main dashboard statistics
type DashboardMetrics struct {
	// Core Stats (replacing current ones)
	TotalInstalls     MetricWithTrend `json:"total_installs"`
	TotalAPICalls     MetricWithTrend `json:"total_api_calls"`
	ActiveUsers       MetricWithTrend `json:"active_users"`
	ServerHealth      MetricWithTrend `json:"server_health"` // Replaces avg usage time
	
	// Additional Key Metrics
	NewServersToday   int64           `json:"new_servers_today"`
	InstallVelocity   float64         `json:"install_velocity"`
	TopRatedCount     int64           `json:"top_rated_count"`
	SearchSuccessRate float64         `json:"search_success_rate"`
	
	// Mini Trends (for sparklines)
	InstallTrend      []int64         `json:"install_trend"`      // Last 7 days
	ActivityTrend     []int64         `json:"activity_trend"`     // Last 7 hours
	
	// Quick Stats
	MostInstalledToday *ServerQuickStat `json:"most_installed_today,omitempty"`
	HottestServer      *ServerQuickStat `json:"hottest_server,omitempty"`
	NewestServer       *ServerQuickStat `json:"newest_server,omitempty"`
}

// MetricWithTrend represents a metric value with its trend
type MetricWithTrend struct {
	Value            interface{} `json:"value"`
	Trend            float64     `json:"trend"`           // Percentage change
	TrendDirection   string      `json:"trend_direction"` // "up", "down", "stable"
	ComparisonPeriod string      `json:"comparison_period"` // "vs yesterday", "vs last week"
}

// ServerQuickStat represents a quick server statistic
type ServerQuickStat struct {
	ServerID   string      `json:"server_id"`
	ServerName string      `json:"server_name"`
	Value      interface{} `json:"value"`
	Label      string      `json:"label"`
}

// DataPoint represents a single data point in time series
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}