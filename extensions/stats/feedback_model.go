package stats

import (
	"time"
)

// ServerFeedback represents a user's rating and comment for a server
type ServerFeedback struct {
	ID           string    `json:"id" bson:"_id"`
	ServerID     string    `json:"server_id" bson:"server_id"`
	Source       string    `json:"source" bson:"source"`
	UserID       string    `json:"user_id" bson:"user_id"`
	Rating       float64   `json:"rating" bson:"rating"`
	Comment      string    `json:"comment,omitempty" bson:"comment,omitempty"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
	IsPublic     bool      `json:"is_public" bson:"is_public"`
	
	// Optional fields for display
	Username     string    `json:"username,omitempty" bson:"-"`
	UserAvatar   string    `json:"user_avatar,omitempty" bson:"-"`
}

// FeedbackResponse for API responses
type FeedbackResponse struct {
	Feedback    []*ServerFeedback `json:"feedback"`
	TotalCount  int              `json:"total_count"`
	HasMore     bool             `json:"has_more"`
}

// FeedbackSortOrder defines how feedback should be sorted
type FeedbackSortOrder string

const (
	FeedbackSortNewest    FeedbackSortOrder = "newest"
	FeedbackSortOldest    FeedbackSortOrder = "oldest"
	FeedbackSortRatingHigh FeedbackSortOrder = "rating_high"
	FeedbackSortRatingLow  FeedbackSortOrder = "rating_low"
)

// UserFeedbackResponse for checking if user has rated
type UserFeedbackResponse struct {
	HasRated bool            `json:"has_rated"`
	Feedback *ServerFeedback `json:"feedback,omitempty"`
}

// FeedbackUpdateRequest for updating existing feedback
type FeedbackUpdateRequest struct {
	Rating  float64 `json:"rating" validate:"required,min=1,max=5"`
	Comment string  `json:"comment,omitempty" validate:"max=1000"`
	UserID  string  `json:"user_id" validate:"required"`
}