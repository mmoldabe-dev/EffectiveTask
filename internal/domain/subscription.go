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
	StartDate   time.Time  `json:"start_date" db:"start_date"`
	EndDate     *time.Time `json:"end_date" db:"end_date"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}
type CreateSubscriptionDTO struct {
    ServiceName string `json:"service_name" validate:"required"`
    Price       int    `json:"price" validate:"required,numeric,min=0"`
    UserID      string `json:"user_id" validate:"required,uuid"`
    StartDate   string `json:"start_date" validate:"required"` 
    EndDate     string `json:"end_date"`                  
}
