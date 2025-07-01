package elasticsearch

import (
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
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