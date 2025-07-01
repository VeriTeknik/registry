package models

// ServerSearchResult represents a server in search results
type ServerSearchResult struct {
	ServerID      string  `json:"server_id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Score         float64 `json:"score"`
	TotalInstalls int64   `json:"total_installs"`
	Rating        float64 `json:"rating"`
	PackageTypes  []string `json:"package_types"`
}