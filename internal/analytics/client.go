package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client provides methods to track analytics events
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new analytics client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Event represents an analytics event
type Event struct {
	EventType  string                 `json:"event_type"`
	ServerID   string                 `json:"server_id,omitempty"`
	ClientID   string                 `json:"client_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TrackServerView tracks when a server detail page is viewed
func (c *Client) TrackServerView(ctx context.Context, serverID, clientID, sessionID string) error {
	event := Event{
		EventType: "view",
		ServerID:  serverID,
		ClientID:  clientID,
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	}
	return c.track(ctx, event)
}

// TrackSearch tracks search queries
func (c *Client) TrackSearch(ctx context.Context, query string, filters map[string]string, resultCount int, clientID, sessionID string) error {
	metadata := map[string]interface{}{
		"query":        query,
		"result_count": resultCount,
		"timestamp":    time.Now().UTC(),
	}
	
	// Add filters to metadata
	for k, v := range filters {
		metadata["filter_"+k] = v
	}
	
	event := Event{
		EventType: "search",
		ClientID:  clientID,
		SessionID: sessionID,
		Metadata:  metadata,
	}
	return c.track(ctx, event)
}

// TrackPublish tracks server publishing events
func (c *Client) TrackPublish(ctx context.Context, serverID string, isUpdate bool, userID, clientID string) error {
	eventType := "publish_new"
	if isUpdate {
		eventType = "publish_update"
	}
	
	event := Event{
		EventType: eventType,
		ServerID:  serverID,
		ClientID:  clientID,
		UserID:    userID,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	}
	return c.track(ctx, event)
}

// TrackError tracks error events
func (c *Client) TrackError(ctx context.Context, errorType string, statusCode int, endpoint string, clientID, sessionID string) error {
	event := Event{
		EventType: "error",
		ClientID:  clientID,
		SessionID: sessionID,
		Metadata: map[string]interface{}{
			"error_type":  errorType,
			"status_code": statusCode,
			"endpoint":    endpoint,
			"timestamp":   time.Now().UTC(),
		},
	}
	return c.track(ctx, event)
}

// track sends the event to the analytics API
func (c *Client) track(ctx context.Context, event Event) error {
	// Skip tracking if no base URL is configured
	if c.baseURL == "" {
		return nil
	}
	
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/track", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Send async to not block the main request
	go func() {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Log error but don't fail the main request
			fmt.Printf("Analytics tracking error: %v\n", err)
			return
		}
		defer resp.Body.Close()
	}()
	
	return nil
}