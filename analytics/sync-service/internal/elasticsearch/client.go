package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/modelcontextprotocol/registry/analytics/sync-service/internal/models"
)

// Client wraps Elasticsearch operations
type Client struct {
	es *elasticsearch.Client
}

// NewClient creates a new Elasticsearch client
func NewClient(addresses ...string) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Test connection
	res, err := es.Info()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error connecting to Elasticsearch: %s", res.String())
	}

	return &Client{es: es}, nil
}

// InitializeIndices creates indices if they don't exist
func (c *Client) InitializeIndices() error {
	indices := []string{"servers", "events", "metrics", "feedback"}
	
	for _, index := range indices {
		exists, err := c.indexExists(index)
		if err != nil {
			return err
		}
		
		if !exists {
			if err := c.createIndex(index); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// IndexServer indexes or updates a server document
func (c *Client) IndexServer(ctx context.Context, server *models.ServerDetail) error {
	// Convert to Elasticsearch format
	esServer := models.ElasticsearchServer{
		ServerID:    server.ID,
		Name:        server.Name,
		Description: server.Description,
		Repository:  server.Repository,
		Version:     server.VersionDetail.Version,
		ReleaseDate: server.VersionDetail.ReleaseDate,
		IsLatest:    server.VersionDetail.IsLatest,
		Packages:    server.Packages,
		IndexedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	// Extract categories and tags (placeholder logic)
	esServer.Categories = extractCategories(server)
	esServer.Tags = extractTags(server)

	// Marshal to JSON
	data, err := json.Marshal(esServer)
	if err != nil {
		return err
	}

	// Index the document
	req := esapi.IndexRequest{
		Index:      "servers",
		DocumentID: server.ID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing server %s: %s", server.ID, res.String())
	}

	return nil
}

// DeleteServer removes a server from the index
func (c *Client) DeleteServer(ctx context.Context, serverID string) error {
	req := esapi.DeleteRequest{
		Index:      "servers",
		DocumentID: serverID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("error deleting server %s: %s", serverID, res.String())
	}

	return nil
}

// BulkIndex performs bulk indexing of multiple servers
func (c *Client) BulkIndex(ctx context.Context, servers []models.ServerDetail) error {
	if len(servers) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for _, server := range servers {
		// Create action metadata
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": "servers",
				"_id":    server.ID,
			},
		}
		metaJSON, _ := json.Marshal(meta)
		buf.Write(metaJSON)
		buf.WriteByte('\n')

		// Create document
		esServer := models.ElasticsearchServer{
			ServerID:    server.ID,
			Name:        server.Name,
			Description: server.Description,
			Repository:  server.Repository,
			Version:     server.VersionDetail.Version,
			ReleaseDate: server.VersionDetail.ReleaseDate,
			IsLatest:    server.VersionDetail.IsLatest,
			Packages:    server.Packages,
			Categories:  extractCategories(&server),
			Tags:        extractTags(&server),
			IndexedAt:   time.Now(),
			LastUpdated: time.Now(),
		}
		docJSON, _ := json.Marshal(esServer)
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body:    &buf,
		Refresh: "true",
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk index error: %s", res.String())
	}

	return nil
}

func (c *Client) indexExists(name string) (bool, error) {
	res, err := c.es.Indices.Exists([]string{name})
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	
	return res.StatusCode == 200, nil
}

func (c *Client) createIndex(name string) error {
	// Basic index creation - in production, load mappings from files
	res, err := c.es.Indices.Create(name)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("error creating index %s: %s", name, res.String())
	}
	
	return nil
}

// Helper functions to extract metadata
func extractCategories(server *models.ServerDetail) []string {
	categories := []string{}
	
	// Extract from name or description
	name := strings.ToLower(server.Name)
	desc := strings.ToLower(server.Description)
	
	if strings.Contains(name, "database") || strings.Contains(desc, "database") {
		categories = append(categories, "database")
	}
	if strings.Contains(name, "api") || strings.Contains(desc, "api") {
		categories = append(categories, "api")
	}
	if strings.Contains(name, "ai") || strings.Contains(desc, "ai") || 
	   strings.Contains(name, "llm") || strings.Contains(desc, "llm") {
		categories = append(categories, "ai")
	}
	
	return categories
}

func extractTags(server *models.ServerDetail) []string {
	tags := []string{}
	
	// Add package registry as tag
	for _, pkg := range server.Packages {
		if pkg.RegistryName != "" && pkg.RegistryName != "unknown" {
			tags = append(tags, pkg.RegistryName)
		}
	}
	
	// Add repository source as tag
	if server.Repository.Source != "" {
		tags = append(tags, server.Repository.Source)
	}
	
	return tags
}