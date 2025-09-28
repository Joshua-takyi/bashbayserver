package models

import (
	"time"

	"github.com/google/uuid"
)

type Bookings struct {
	ID         uuid.UUID `db:"id" json:"id"`
	EventId    uuid.UUID `db:"event_id" json:"event_id"`
	VenueId    uuid.UUID `db:"venues_id" json:"venues_id"`
	UserId     uuid.UUID `db:"user_id" json:"user_id"`
	StartTime  time.Time `db:"start_time" json:"start_time"`
	EndTime    time.Time `db:"end_time" json:"end_time"`
	TotalPrice float64   `db:"total_price" json:"total_price"`
	// status to track booking state (e.g., "pending", "confirmed", "canceled")
	Status        string    `db:"status" json:"status"`
	PaymentStatus string    `db:"payment_status" json:"payment_status"` // eg "pending", "paid", "failed"
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}
