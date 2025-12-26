package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          int64     `json:"id" example:"10"`
	UserID      uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ServiceName string    `json:"service_name" example:"Spotify Premium"`
	Price       int       `json:"price" example:"500"`
	StartDate   string    `json:"start_date" example:"01-2026"`
	EndDate     *string   `json:"end_date,omitempty" example:"12-2026"`
	CreatedAt   time.Time `json:"created_at,omitempty" swaggerignore:"true"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" swaggerignore:"true"`
}

type SubscriptionFilter struct {
	UserID      uuid.UUID
	ServiceName string
	MinPrice    int
	MaxPrice    int

	Limit  int
	Offset int
}
