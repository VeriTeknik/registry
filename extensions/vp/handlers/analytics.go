package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/registry/extensions/stats"
)

// GetDashboardMetricsHandler returns the main dashboard metrics
func (h *VPHandlers) GetDashboardMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Debug log
	log.Printf("GetDashboardMetricsHandler called - method: %s, path: %s", r.Method, r.URL.Path)
	
	// Get time period from query params
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	// Check cache
	cacheKey := fmt.Sprintf("vp:dashboard:%s", period)
	if cached, found := h.statsCache.Get(cacheKey); found {
		WriteCachedResponse(w, cached, true)
		return
	}

	// Get analytics metrics
	metrics, err := h.analyticsDB.GetAnalyticsMetrics(r.Context(), period)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get dashboard metrics: %v", err))
		return
	}

	// Get trending data
	trending, _ := h.analyticsDB.CalculateTrending(r.Context(), 1)
	
	// Get recent activity for sparklines
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour)
	timeSeries, _ := h.analyticsDB.GetTimeSeries(r.Context(), startTime, endTime, "day")
	
	// Build install trend
	installTrend := make([]int64, 7)
	for i, ts := range timeSeries {
		if i < 7 {
			installTrend[i] = ts.Installs
		}
	}

	// Get comparison data for trends
	var previousMetrics *stats.AnalyticsMetrics
	if period == "day" {
		yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
		previousMetrics, _ = h.analyticsDB.GetAnalyticsMetrics(r.Context(), yesterday)
	}

	// Calculate trends
	installsTrend := calculateTrend(metrics.TotalInstalls, previousMetrics.TotalInstalls)
	apiCallsTrend := calculateTrend(metrics.TotalAPICalls, previousMetrics.TotalAPICalls)
	activeUsersTrend := calculateTrend(metrics.ActiveUsers, previousMetrics.ActiveUsers)
	
	// Calculate server health score (based on uptime and response time)
	healthScore := calculateHealthScore(metrics.UptimePercentage, metrics.ResponseTimeP50)
	var previousHealthScore float64
	if previousMetrics != nil {
		previousHealthScore = calculateHealthScore(previousMetrics.UptimePercentage, previousMetrics.ResponseTimeP50)
	}
	healthTrend := calculateTrendFloat(healthScore, previousHealthScore)

	// Get hot server
	var hottestServer *stats.ServerQuickStat
	if len(trending) > 0 {
		hottestServer = &stats.ServerQuickStat{
			ServerID:   trending[0].ServerID,
			ServerName: trending[0].ServerName,
			Value:      fmt.Sprintf("%.1f/hr", trending[0].InstallVelocity),
			Label:      "installs/hour",
		}
	}

	// Get newest server
	recentServers, _ := h.statsDB.GetRecentServers(r.Context(), 1, "")
	var newestServer *stats.ServerQuickStat
	if len(recentServers) > 0 {
		server, _ := h.service.GetByID(recentServers[0].ServerID)
		if server != nil {
			newestServer = &stats.ServerQuickStat{
				ServerID:   server.ID,
				ServerName: server.Name,
				Value:      "Just added",
				Label:      time.Since(recentServers[0].FirstSeen).Round(time.Minute).String() + " ago",
			}
		}
	}

	// Build response
	dashboard := &stats.DashboardMetrics{
		TotalInstalls: stats.MetricWithTrend{
			Value:            metrics.TotalInstalls,
			Trend:            installsTrend,
			TrendDirection:   getTrendDirection(installsTrend),
			ComparisonPeriod: getComparisonPeriod(period),
		},
		TotalAPICalls: stats.MetricWithTrend{
			Value:            metrics.TotalAPICalls,
			Trend:            apiCallsTrend,
			TrendDirection:   getTrendDirection(apiCallsTrend),
			ComparisonPeriod: getComparisonPeriod(period),
		},
		ActiveUsers: stats.MetricWithTrend{
			Value:            metrics.ActiveUsers,
			Trend:            activeUsersTrend,
			TrendDirection:   getTrendDirection(activeUsersTrend),
			ComparisonPeriod: getComparisonPeriod(period),
		},
		ServerHealth: stats.MetricWithTrend{
			Value:            fmt.Sprintf("%.1f%%", healthScore),
			Trend:            healthTrend,
			TrendDirection:   getTrendDirection(healthTrend),
			ComparisonPeriod: getComparisonPeriod(period),
		},
		NewServersToday:    metrics.NewServers,
		InstallVelocity:    metrics.InstallVelocity,
		TopRatedCount:      metrics.FiveStarServers,
		SearchSuccessRate:  metrics.SearchSuccessRate,
		InstallTrend:       installTrend,
		HottestServer:      hottestServer,
		NewestServer:       newestServer,
	}

	// Cache the response
	h.statsCache.Set(cacheKey, dashboard)

	// Send response
	if err := WriteCachedResponse(w, dashboard, false); err != nil {
		WriteStandardError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// GetAnalyticsHandler returns comprehensive analytics data
func (h *VPHandlers) GetAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	// Get parameters
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	includeActivity := r.URL.Query().Get("include_activity") == "true"
	includeTrending := r.URL.Query().Get("include_trending") == "true"
	includeCategories := r.URL.Query().Get("include_categories") == "true"
	includeSearch := r.URL.Query().Get("include_search") == "true"

	// Get base metrics
	metrics, err := h.analyticsDB.GetAnalyticsMetrics(r.Context(), period)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get analytics: %v", err))
		return
	}

	response := &stats.AnalyticsResponse{
		Metrics:      metrics,
		TimePeriod:   period,
		GeneratedAt:  time.Now(),
	}

	// Add optional data
	if includeTrending {
		trending, _ := h.analyticsDB.CalculateTrending(r.Context(), 10)
		response.TrendingServers = trending
	}

	if includeActivity {
		activity, _ := h.analyticsDB.GetRecentActivity(r.Context(), 20, "")
		response.RecentActivity = activity
	}

	if includeCategories {
		categories, _ := h.analyticsDB.GetCategoryStats(r.Context())
		response.CategoryBreakdown = categories
	}

	if includeSearch {
		searches, _ := h.analyticsDB.GetTopSearches(r.Context(), 10)
		response.SearchInsights = searches
	}

	// Get recent milestones
	milestones, _ := h.analyticsDB.GetRecentMilestones(r.Context(), 5)
	if len(milestones) > 0 {
		response.Milestones = milestones
	}

	// Send response
	writeJSONResponse(w, response)
}

// GetActivityFeedHandler returns real-time activity feed
func (h *VPHandlers) GetActivityFeedHandler(w http.ResponseWriter, r *http.Request) {
	// Parse parameters
	limitStr := r.URL.Query().Get("limit")
	limit := parseLimit(limitStr, 20, 100)
	
	eventType := r.URL.Query().Get("type")
	
	// Get recent activity
	activity, err := h.analyticsDB.GetRecentActivity(r.Context(), limit, eventType)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get activity: %v", err))
		return
	}

	// Enrich activity with server names
	for i := range activity {
		if activity[i].ServerID != "" && activity[i].ServerName == "" {
			server, err := h.service.GetByID(activity[i].ServerID)
			if err == nil {
				activity[i].ServerName = server.Name
			}
		}
	}

	response := map[string]interface{}{
		"activity": activity,
		"count":    len(activity),
		"type":     eventType,
	}

	writeJSONResponse(w, response)
}

// GetGrowthMetricsHandler returns growth statistics
func (h *VPHandlers) GetGrowthMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "installs"
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	growth, err := h.analyticsDB.GetGrowthMetrics(r.Context(), metric, period)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get growth metrics: %v", err))
		return
	}

	writeJSONResponse(w, growth)
}

// GetAPIMetricsHandler returns API usage statistics
func (h *VPHandlers) GetAPIMetricsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := parseLimit(limitStr, 20, 100)

	metrics, err := h.analyticsDB.GetAPIMetrics(r.Context(), limit)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get API metrics: %v", err))
		return
	}

	response := map[string]interface{}{
		"endpoints": metrics,
		"count":     len(metrics),
	}

	writeJSONResponse(w, response)
}

// GetSearchAnalyticsHandler returns search analytics
func (h *VPHandlers) GetSearchAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := parseLimit(limitStr, 20, 100)

	searches, err := h.analyticsDB.GetTopSearches(r.Context(), limit)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get search analytics: %v", err))
		return
	}

	// Calculate overall search success rate
	var totalSearches, totalConversions int64
	for _, s := range searches {
		totalSearches += s.Count
		totalConversions += s.InstallsFromSearch
	}
	
	overallSuccessRate := float64(0)
	if totalSearches > 0 {
		overallSuccessRate = float64(totalConversions) / float64(totalSearches) * 100
	}

	response := map[string]interface{}{
		"top_searches":         searches,
		"total_searches":       totalSearches,
		"overall_success_rate": overallSuccessRate,
	}

	writeJSONResponse(w, response)
}

// GetTimeSeriesHandler returns time series data for charts
func (h *VPHandlers) GetTimeSeriesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse time range
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	interval := r.URL.Query().Get("interval")
	
	if interval == "" {
		interval = "hour"
	}

	// Default to last 7 days if not specified
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour)

	if startStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = parsed
		}
	}

	if endStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = parsed
		}
	}

	// Get time series data
	data, err := h.analyticsDB.GetTimeSeries(r.Context(), startTime, endTime, interval)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get time series: %v", err))
		return
	}

	response := map[string]interface{}{
		"data":      data,
		"start":     startTime,
		"end":       endTime,
		"interval":  interval,
		"count":     len(data),
	}

	writeJSONResponse(w, response)
}

// GetHotServersHandler returns servers with sudden activity spikes
func (h *VPHandlers) GetHotServersHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := parseLimit(limitStr, 10, 50)

	// Get trending servers with high momentum
	trending, err := h.analyticsDB.CalculateTrending(r.Context(), limit)
	if err != nil {
		WriteStandardError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get hot servers: %v", err))
		return
	}

	// Filter for servers with significant momentum change
	var hotServers []stats.TrendingServer
	for _, server := range trending {
		if server.MomentumChange > 50 { // 50% increase in velocity
			hotServers = append(hotServers, server)
		}
	}

	response := map[string]interface{}{
		"servers": hotServers,
		"count":   len(hotServers),
	}

	writeJSONResponse(w, response)
}

// TrackAPICallMiddleware middleware to track API calls
func (h *VPHandlers) TrackAPICallMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Call the next handler
		next(wrapped, r)
		
		// Track the API call
		duration := time.Since(start).Milliseconds()
		isError := wrapped.statusCode >= 400
		
		endpoint := r.URL.Path
		method := r.Method
		
		// Track asynchronously to not block response
		go func() {
			if err := h.analyticsDB.TrackAPICall(r.Context(), endpoint, method, float64(duration), isError); err != nil {
				log.Printf("Failed to track API call: %v", err)
			}
		}()
	}
}

// Helper functions

func calculateTrend(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return float64(current-previous) / float64(previous) * 100
}

func calculateTrendFloat(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return (current - previous) / previous * 100
}

func getTrendDirection(trend float64) string {
	if trend > 0 {
		return "up"
	} else if trend < 0 {
		return "down"
	}
	return "stable"
}

func getComparisonPeriod(period string) string {
	switch period {
	case "day":
		return "vs yesterday"
	case "week":
		return "vs last week"
	case "month":
		return "vs last month"
	default:
		return "vs previous period"
	}
}

func calculateHealthScore(uptime, responseTime float64) float64 {
	// Simple health score: uptime percentage - (response time penalty)
	// Response time over 100ms starts reducing health score
	responsePenalty := 0.0
	if responseTime > 100 {
		responsePenalty = (responseTime - 100) / 10 // -1% for every 10ms over 100ms
	}
	
	health := uptime - responsePenalty
	if health < 0 {
		health = 0
	}
	return health
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}