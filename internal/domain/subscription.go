package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID          int64      `json:"id" db:"id"`
	ServiceName string     `json:"service_name" db:"service_name"`
	Price       int        `json:"price" db:"price"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	StartDate   string  `json:"start_date" db:"start_date"`
	EndDate     *string `json:"end_date,omitempty" db:"end_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type SubscriptionFilter struct {
	UserID      uuid.UUID
	ServiceName string
	MinPrice    int
	MaxPrice    int

	Limit  int
	Offset int
}


