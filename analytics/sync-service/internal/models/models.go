package models

import (
	"time"
)

// Server represents a server document from MongoDB
type Server struct {
	ID            string        `bson:"id" json:"server_id"`
	Name          string        `bson:"name" json:"name"`
	Description   string        `bson:"description" json:"description"`
	Repository    Repository    `bson:"repository" json:"repository"`
	VersionDetail VersionDetail `bson:"version_detail" json:"version_detail"`
}

// ServerDetail includes packages and remotes
type ServerDetail struct {
	Server   `bson:",inline"`
	Packages []Package `bson:"packages" json:"packages"`
	Remotes  []Remote  `bson:"remotes" json:"remotes"`
}

// Repository represents repository information
type Repository struct {
	URL    string `bson:"url" json:"url"`
	Source string `bson:"source" json:"source"`
	ID     string `bson:"id" json:"id"`
}

// VersionDetail represents version information
type VersionDetail struct {
	Version     string    `bson:"version" json:"version"`
	ReleaseDate time.Time `bson:"release_date" json:"release_date"`
	IsLatest    bool      `bson:"is_latest" json:"is_latest"`
}

// Package represents a package
type Package struct {
	RegistryName string `bson:"registry_name" json:"registry_name"`
	Name         string `bson:"name" json:"name"`
	Version      string `bson:"version" json:"version"`
}

// Remote represents a remote endpoint
type Remote struct {
	TransportType string `bson:"transport_type" json:"transport_type"`
	URL           string `bson:"url" json:"url"`
}

// ElasticsearchServer represents a server document for Elasticsearch
type ElasticsearchServer struct {
	ServerID    string        `json:"server_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Repository  Repository    `json:"repository"`
	Version     string        `json:"version"`
	ReleaseDate time.Time     `json:"release_date"`
	IsLatest    bool          `json:"is_latest"`
	Packages    []Package     `json:"packages"`
	Categories  []string      `json:"categories,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	IndexedAt   time.Time     `json:"indexed_at"`
	LastUpdated time.Time     `json:"last_updated"`
}