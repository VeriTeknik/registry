package service

import (
	"context"
	"fmt"
	"time"
	
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/elasticsearch"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/models"
	"github.com/modelcontextprotocol/registry/analytics/analytics-api/internal/redis"
)

// AnalyticsService handles analytics operations
type AnalyticsService struct {
	es    *elasticsearch.Client
	redis *redis.Client
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(es *elasticsearch.Client, redis *redis.Client) *AnalyticsService {
	return &AnalyticsService{
		es:    es,
		redis: redis,
	}
}

// TrackEvent tracks an analytics event
func (s *AnalyticsService) TrackEvent(ctx context.Context, event *models.Event) error {
	// Enrich event with server name from servers index
	if event.ServerID != "" {
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"server_id": event.ServerID,
				},
			},
			"_source": []string{"name"},
		}
		
		result, err := s.es.Search(ctx, "servers", query)
		if err == nil {
			if hits, ok := result["hits"].(map[string]interface{}); ok {
				if hitsList, ok := hits["hits"].([]interface{}); ok && len(hitsList) > 0 {
					if hit, ok := hitsList[0].(map[string]interface{}); ok {
						if source, ok := hit["_source"].(map[string]interface{}); ok {
							event.ServerName = getStringValue(source, "name")
						}
					}
				}
			}
		}
	}
	
	// Index the event
	err := s.es.Index(ctx, "events", event)
	if err != nil {
		return fmt.Errorf("failed to index event: %w", err)
	}
	
	// Update Redis counters for real-time metrics (if needed)
	// This could include daily counters, etc.
	
	return nil
}

// GetServerStats returns server statistics
func (s *AnalyticsService) GetServerStats(ctx context.Context, serverID string) (*models.ServerStats, error) {
	// Get server info
	serverQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"server_id": serverID,
			},
		},
		"_source": []string{"name"},
	}
	
	var serverName string
	serverResult, err := s.es.Search(ctx, "servers", serverQuery)
	if err == nil {
		if hits, ok := serverResult["hits"].(map[string]interface{}); ok {
			if hitsList, ok := hits["hits"].([]interface{}); ok && len(hitsList) > 0 {
				if hit, ok := hitsList[0].(map[string]interface{}); ok {
					if source, ok := hit["_source"].(map[string]interface{}); ok {
						serverName = getStringValue(source, "name")
					}
				}
			}
		}
	}
	
	// Count total installs
	installQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": []map[string]interface{}{
				{"term": map[string]interface{}{"server_id": serverID}},
				{"term": map[string]interface{}{"event_type": "install"}},
			},
		},
	}
	totalInstalls, _ := s.es.Count(ctx, "events", installQuery)
	
	// Count total usage events
	usageQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": []map[string]interface{}{
				{"term": map[string]interface{}{"server_id": serverID}},
				{"term": map[string]interface{}{"event_type": "usage"}},
			},
		},
	}
	totalUsage, _ := s.es.Count(ctx, "events", usageQuery)
	
	// Calculate daily active users (unique client_ids in last 24h)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	dauQuery := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{"term": map[string]interface{}{"server_id": serverID}},
					{"range": map[string]interface{}{
						"timestamp": map[string]interface{}{
							"gte": yesterday.Format(time.RFC3339),
						},
					}},
				},
			},
		},
		"aggs": map[string]interface{}{
			"unique_users": map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": "client_id",
				},
			},
		},
	}
	
	var dau int64 = 0
	dauResult, err := s.es.Search(ctx, "events", dauQuery)
	if err == nil {
		if aggs, ok := dauResult["aggregations"].(map[string]interface{}); ok {
			if users, ok := aggs["unique_users"].(map[string]interface{}); ok {
				if value, ok := users["value"].(float64); ok {
					dau = int64(value)
				}
			}
		}
	}
	
	// Calculate monthly active users (unique client_ids in last 30 days)
	thirtyDaysAgo := now.Add(-30 * 24 * time.Hour)
	mauQuery := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{"term": map[string]interface{}{"server_id": serverID}},
					{"range": map[string]interface{}{
						"timestamp": map[string]interface{}{
							"gte": thirtyDaysAgo.Format(time.RFC3339),
						},
					}},
				},
			},
		},
		"aggs": map[string]interface{}{
			"unique_users": map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": "client_id",
				},
			},
		},
	}
	
	var mau int64 = 0
	mauResult, err := s.es.Search(ctx, "events", mauQuery)
	if err == nil {
		if aggs, ok := mauResult["aggregations"].(map[string]interface{}); ok {
			if users, ok := aggs["unique_users"].(map[string]interface{}); ok {
				if value, ok := users["value"].(float64); ok {
					mau = int64(value)
				}
			}
		}
	}
	
	return &models.ServerStats{
		ServerID:           serverID,
		ServerName:         serverName,
		TotalInstalls:      totalInstalls,
		ActiveInstalls:     totalInstalls, // Would need uninstall tracking to calculate properly
		TotalUsage:         totalUsage,
		DailyActiveUsers:   dau,
		MonthlyActiveUsers: mau,
		AverageRating:      0, // Will be calculated when ratings are implemented
		RatingCount:        0,
		CommentCount:       0,
		LastUpdated:        time.Now(),
	}, nil
}

// GetServerTimeline returns timeline data
func (s *AnalyticsService) GetServerTimeline(ctx context.Context, serverID string, period string) ([]models.TimelineData, error) {
	// Parse period (default to 30d)
	days := 30
	if period != "" {
		if len(period) > 1 && period[len(period)-1] == 'd' {
			if d, err := fmt.Sscanf(period[:len(period)-1], "%d", &days); err == nil && d > 0 {
				days = d
			}
		}
	}
	
	// Calculate date range
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(days) * 24 * time.Hour)
	
	// Query for aggregated data by date
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{"term": map[string]interface{}{"server_id": serverID}},
					{"range": map[string]interface{}{
						"timestamp": map[string]interface{}{
							"gte": startDate.Format(time.RFC3339),
							"lte": endDate.Format(time.RFC3339),
						},
					}},
				},
			},
		},
		"aggs": map[string]interface{}{
			"timeline": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field": "timestamp",
					"calendar_interval": "1d",
					"format": "yyyy-MM-dd",
					"min_doc_count": 0,
					"extended_bounds": map[string]interface{}{
						"min": startDate.Format("2006-01-02"),
						"max": endDate.Format("2006-01-02"),
					},
				},
				"aggs": map[string]interface{}{
					"events_by_type": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "event_type",
						},
					},
					"unique_users": map[string]interface{}{
						"cardinality": map[string]interface{}{
							"field": "client_id",
						},
					},
				},
			},
		},
	}
	
	result, err := s.es.Search(ctx, "events", query)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline data: %w", err)
	}
	
	timeline := []models.TimelineData{}
	
	if aggs, ok := result["aggregations"].(map[string]interface{}); ok {
		if timelineAgg, ok := aggs["timeline"].(map[string]interface{}); ok {
			if buckets, ok := timelineAgg["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						date := getStringValue(b, "key_as_string")
						
						var installs, usage, errors, users int64
						
						// Extract unique users
						if uniqueUsers, ok := b["unique_users"].(map[string]interface{}); ok {
							if value, ok := uniqueUsers["value"].(float64); ok {
								users = int64(value)
							}
						}
						
						// Extract event counts by type
						if eventsByType, ok := b["events_by_type"].(map[string]interface{}); ok {
							if eventBuckets, ok := eventsByType["buckets"].([]interface{}); ok {
								for _, eventBucket := range eventBuckets {
									if eb, ok := eventBucket.(map[string]interface{}); ok {
										eventType := getStringValue(eb, "key")
										if count, ok := eb["doc_count"].(float64); ok {
											switch eventType {
											case "install":
												installs = int64(count)
											case "usage":
												usage = int64(count)
											case "error":
												errors = int64(count)
											}
										}
									}
								}
							}
						}
						
						timeline = append(timeline, models.TimelineData{
							Date:     date,
							Installs: installs,
							Usage:    usage,
							Errors:   errors,
							Users:    users,
						})
					}
				}
			}
		}
	}
	
	return timeline, nil
}

// GetTrending returns trending servers
func (s *AnalyticsService) GetTrending(ctx context.Context, period string, limit int) ([]models.TrendingServer, error) {
	// Since we don't have events data yet, we'll use recently updated servers
	// In a real implementation, this would calculate trending based on event activity
	
	// Query for recently updated servers
	query := map[string]interface{}{
		"size": limit,
		"sort": []map[string]interface{}{
			{"last_updated": map[string]string{"order": "desc"}},
		},
		"_source": []string{"server_id", "name", "description", "last_updated"},
	}

	result, err := s.es.Search(ctx, "servers", query)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending servers: %w", err)
	}

	var trendingServers []models.TrendingServer
	if hits, ok := result["hits"].(map[string]interface{}); ok {
		if hitsList, ok := hits["hits"].([]interface{}); ok {
			for i, hit := range hitsList {
				if h, ok := hit.(map[string]interface{}); ok {
					if source, ok := h["_source"].(map[string]interface{}); ok {
						server := models.TrendingServer{
							ServerID:       getStringValue(source, "server_id"),
							ServerName:     getStringValue(source, "name"),
							Description:    getStringValue(source, "description"),
							TrendingScore:  float64(limit - i), // Simple score based on recency
							InstallGrowth:  0, // Will be calculated once we have events
							UsageGrowth:    0, // Will be calculated once we have events
							RecentInstalls: 0, // Will be calculated once we have events
						}
						trendingServers = append(trendingServers, server)
					}
				}
			}
		}
	}

	// Once we have events data, we would:
	// 1. Aggregate events by server_id for the period
	// 2. Compare with previous period to calculate growth
	// 3. Sort by trending score (combination of absolute numbers and growth rate)
	
	return trendingServers, nil
}

// Helper function to safely get string values
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetPopular returns popular servers
func (s *AnalyticsService) GetPopular(ctx context.Context, category string, limit int) ([]models.TrendingServer, error) {
	// TODO: Implement popular servers
	return []models.TrendingServer{}, nil
}

// SearchServers searches for servers
func (s *AnalyticsService) SearchServers(ctx context.Context, req *models.SearchRequest) (*models.SearchResult, error) {
	// Build search query
	must := []map[string]interface{}{}
	filter := []map[string]interface{}{}
	
	// Add search query if provided
	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": req.Query,
				"fields": []string{"name^3", "description^2", "tags"},
				"type": "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}
	
	// Add category filter if provided
	if len(req.Categories) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"tags.keyword": req.Categories,
			},
		})
	}
	
	// Add package type filter if provided
	if len(req.PackageTypes) > 0 {
		filter = append(filter, map[string]interface{}{
			"terms": map[string]interface{}{
				"packages.keyword": req.PackageTypes,
			},
		})
	}
	
	// Build the query
	query := map[string]interface{}{
		"from": req.Offset,
		"size": req.Limit,
		"track_total_hits": true,
	}
	
	// Add search conditions
	if len(must) > 0 || len(filter) > 0 {
		boolQuery := map[string]interface{}{}
		if len(must) > 0 {
			boolQuery["must"] = must
		}
		if len(filter) > 0 {
			boolQuery["filter"] = filter
		}
		query["query"] = map[string]interface{}{"bool": boolQuery}
	}
	
	// Add sorting
	switch req.SortBy {
	case "relevance":
		// Default ES scoring
	case "name":
		query["sort"] = []map[string]interface{}{
			{"name.keyword": map[string]string{"order": "asc"}},
		}
	case "updated":
		query["sort"] = []map[string]interface{}{
			{"last_updated": map[string]string{"order": "desc"}},
		}
	default:
		// Default to relevance
	}
	
	// Execute search
	start := time.Now()
	result, err := s.es.Search(ctx, "servers", query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	took := time.Since(start).Milliseconds()
	
	// Parse results
	servers := []models.ServerSearchResult{}
	var total int64 = 0
	
	if hits, ok := result["hits"].(map[string]interface{}); ok {
		// Get total count
		if totalHits, ok := hits["total"].(map[string]interface{}); ok {
			if value, ok := totalHits["value"].(float64); ok {
				total = int64(value)
			}
		}
		
		// Parse hits
		if hitsList, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				if h, ok := hit.(map[string]interface{}); ok {
					if source, ok := h["_source"].(map[string]interface{}); ok {
						var score float64 = 0
						if s, ok := h["_score"].(float64); ok {
							score = s
						}
						
						// Extract packages array
						packages := []string{}
						if pkgs, ok := source["packages"].([]interface{}); ok {
							for _, pkg := range pkgs {
								if p, ok := pkg.(string); ok {
									packages = append(packages, p)
								}
							}
						}
						
						server := models.ServerSearchResult{
							ServerID:      getStringValue(source, "server_id"),
							Name:          getStringValue(source, "name"),
							Description:   getStringValue(source, "description"),
							Score:         score,
							TotalInstalls: 0, // Would need to aggregate from events
							Rating:        0, // Would need to calculate from ratings
							PackageTypes:  packages,
						}
						servers = append(servers, server)
					}
				}
			}
		}
	}
	
	return &models.SearchResult{
		Servers:    servers,
		TotalCount: total,
		Took:       took,
	}, nil
}

// RateServer saves a server rating
func (s *AnalyticsService) RateServer(ctx context.Context, rating *models.Rating) error {
	// TODO: Implement rating
	return nil
}

// GetRatings returns server ratings
func (s *AnalyticsService) GetRatings(ctx context.Context, serverID string, limit, offset int) ([]models.Rating, int64, error) {
	// TODO: Implement ratings retrieval
	return []models.Rating{}, 0, nil
}

// AddComment adds a comment
func (s *AnalyticsService) AddComment(ctx context.Context, comment *models.Comment) (string, error) {
	// TODO: Implement comment creation
	return "comment-id", nil
}

// GetComments returns server comments
func (s *AnalyticsService) GetComments(ctx context.Context, serverID string, limit, offset int) ([]models.Comment, int64, error) {
	// TODO: Implement comments retrieval
	return []models.Comment{}, 0, nil
}

// GetGlobalMetrics returns global analytics metrics
func (s *AnalyticsService) GetGlobalMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Get total servers count
	serverCount, err := s.es.Count(ctx, "servers", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count servers: %w", err)
	}

	// Get events count for today
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eventsQuery := map[string]interface{}{
		"range": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"gte": startOfDay.Format(time.RFC3339),
			},
		},
	}
	eventsToday, err := s.es.Count(ctx, "events", eventsQuery)
	if err != nil {
		// If events index doesn't exist or is empty, use 0
		eventsToday = 0
	}

	// Get popular tags from servers (using tags instead of categories)
	categoriesAgg := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "tags.keyword",
					"size": 10,
				},
			},
		},
	}

	catResult, err := s.es.Search(ctx, "servers", categoriesAgg)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Initialize empty slice instead of nil
	popularCategories := []map[string]interface{}{}
	if aggs, ok := catResult["aggregations"].(map[string]interface{}); ok {
		if tags, ok := aggs["tags"].(map[string]interface{}); ok {
			if buckets, ok := tags["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						cat := map[string]interface{}{
							"name":  b["key"],
							"count": b["doc_count"],
						}
						popularCategories = append(popularCategories, cat)
					}
				}
			}
		}
	}

	// Calculate active users from events (unique client_ids in last 24h)
	last24h := time.Now().Add(-24 * time.Hour)
	activeUsersQuery := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"gte": last24h.Format(time.RFC3339),
				},
			},
		},
		"aggs": map[string]interface{}{
			"unique_users": map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": "client_id",
				},
			},
		},
	}

	var activeUsers int64 = 0
	activeResult, err := s.es.Search(ctx, "events", activeUsersQuery)
	if err == nil {
		if aggs, ok := activeResult["aggregations"].(map[string]interface{}); ok {
			if users, ok := aggs["unique_users"].(map[string]interface{}); ok {
				if value, ok := users["value"].(float64); ok {
					activeUsers = int64(value)
				}
			}
		}
	}

	// Get total installs (count of install events)
	installsQuery := map[string]interface{}{
		"match": map[string]interface{}{
			"event_type": "install",
		},
	}
	totalInstalls, err := s.es.Count(ctx, "events", installsQuery)
	if err != nil {
		totalInstalls = 0
	}

	return map[string]interface{}{
		"total_servers":       serverCount,
		"total_installs":      totalInstalls,
		"active_users":        activeUsers,
		"events_today":        eventsToday,
		"popular_categories":  popularCategories, // Using tags but keeping field name for API compatibility
		"popular_tags":        popularCategories, // Also include as popular_tags
	}, nil
}