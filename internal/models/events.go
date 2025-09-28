package models

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID uuid.UUID `db:"id" json:"id"`

	VenueId      uuid.UUID `db:"venue_id" json:"venue_id"`
	HostId       uuid.UUID `db:"host_id" json:"host_id"`
	Title        string    `db:"title" json:"title"`                 // e.g., "Birthday Party"
	Description  string    `db:"description" json:"description"`     // e.g., "A fun birthday celebration"
	StartTime    time.Time `db:"start_time" json:"start_time"`       // e.g., "2023-10-01T18:00:00Z"
	EndTime      time.Time `db:"end_time" json:"end_time"`           // e.g., "2023-10-01T21:00:00Z"
	MaxAttendees int       `db:"max_attendees" json:"max_attendees"` // e.g., 50
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
