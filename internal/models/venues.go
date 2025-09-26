package models

import (
	"time"

	"github.com/google/uuid"
)

type Venues struct {
	Id          uuid.UUID `db:"id" json:"id"`
	HostId      uuid.UUID `db:"host_id" json:"host_id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"` // e.g., "A beautiful venue for events"
	Location    string    `db:"location" json:"location"`       // e.g., "123 Main St, City, Country"
	Coordinates struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `db:"coordinates" json:"coordinates"` // e.g., {"latitude": 40.7128, "longitude": -74.0060}

	Capacity     int                    `db:"capacity" json:"capacity"`             // e.g., 100
	Amenities    map[string]interface{} `db:"amenities" json:"amenities"`           // e.g., {"wifi": true, "parking": false}
	PricePerHour float64                `db:"price_per_hour" json:"price_per_hour"` // e.g., 50.00
	Availability map[string]interface{} `db:"availability" json:"availability"`     // e.g., {"monday": "9am-5pm", ...}
	// status for admin to approve or reject venue
	Status    string    `db:"status" json:"status"` // e.g., "pending"
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
