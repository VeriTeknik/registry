package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
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

// GetClient returns the underlying Elasticsearch client
func (c *Client) GetClient() *elasticsearch.Client {
	return c.es
}

// Index indexes a document
func (c *Client) Index(ctx context.Context, index string, body interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index: index,
		Body:  &buf,
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	return nil
}

// Search performs a search query
func (c *Client) Search(ctx context.Context, index string, query map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(&buf),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("error parsing error response: %s", err.Error())
		}
		return nil, fmt.Errorf("error searching: %v", e)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Count returns the count of documents matching a query
func (c *Client) Count(ctx context.Context, index string, query map[string]interface{}) (int64, error) {
	var buf bytes.Buffer
	if query != nil {
		if err := json.NewEncoder(&buf).Encode(map[string]interface{}{"query": query}); err != nil {
			return 0, err
		}
	}

	res, err := c.es.Count(
		c.es.Count.WithContext(ctx),
		c.es.Count.WithIndex(index),
		c.es.Count.WithBody(&buf),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("error counting documents: %s", res.String())
	}

	var result struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Count, nil
}

// Bulk performs bulk operations
func (c *Client) Bulk(ctx context.Context, body io.Reader) error {
	res, err := c.es.Bulk(body,
		c.es.Bulk.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk operation failed: %s", res.String())
	}

	return nil
}

// CreateBulkIndexer creates a bulk indexer for efficient indexing
func (c *Client) CreateBulkString(action string, index string, doc interface{}) (string, error) {
	var meta, data bytes.Buffer
	
	meta.WriteString(fmt.Sprintf(`{"%s":{"_index":"%s"}}`, action, index))
	meta.WriteString("\n")
	
	if err := json.NewEncoder(&data).Encode(doc); err != nil {
		return "", err
	}
	
	return meta.String() + data.String(), nil
}