package models

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type VenueStatus string

const (
	StatusPending  VenueStatus = "pending"
	StatusActive   VenueStatus = "active"
	StatusInactive VenueStatus = "inactive"
)

// Coordinates maps to PostGIS geography(Point,4326)
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Scan allows Coordinates to be read from Postgres
func (c *Coordinates) Scan(src interface{}) error {
	var dataStr string

	// Handle different input types
	switch v := src.(type) {
	case []byte:
		dataStr = string(v)
	case string:
		dataStr = v
	case nil:
		return fmt.Errorf("coordinates cannot be nil")
	default:
		return fmt.Errorf("cannot scan %T into Coordinates", src)
	}

	// First try WKT formats (for backward compatibility)
	var lon, lat float64
	var err error

	// Try different WKT formats
	_, err = fmt.Sscanf(dataStr, "POINT(%f %f)", &lon, &lat)
	if err == nil {
		c.Latitude = lat
		c.Longitude = lon
		return nil
	}

	_, err = fmt.Sscanf(dataStr, "SRID=4326;POINT(%f %f)", &lon, &lat)
	if err == nil {
		c.Latitude = lat
		c.Longitude = lon
		return nil
	}

	// If WKT parsing failed, try EWKB (hex-encoded binary)
	if len(dataStr) >= 32 { // EWKB for point should be at least 32 hex chars
		// Decode hex string to bytes
		ewkbBytes, err := hex.DecodeString(dataStr)
		if err != nil {
			return fmt.Errorf("failed to decode EWKB hex: %v", err)
		}

		// Parse EWKB binary format
		return c.parseEWKB(ewkbBytes)
	}

	return fmt.Errorf("failed to parse coordinates from format: %s (input: %q)", err.Error(), dataStr)
}

// parseEWKB parses Extended Well-Known Binary format for PostGIS Point
func (c *Coordinates) parseEWKB(data []byte) error {
	if len(data) < 21 {
		return fmt.Errorf("EWKB data too short: %d bytes", len(data))
	}

	// EWKB format for Point with SRID:
	// Byte 0: Endianness (1 = little endian)
	// Bytes 1-4: Type with SRID flag (0x20000001 = Point with SRID)
	// Bytes 5-8: SRID (4326)
	// Bytes 9-16: X coordinate (longitude)
	// Bytes 17-24: Y coordinate (latitude)

	endian := data[0]
	var order binary.ByteOrder
	if endian == 1 {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	// Read type (should be 0x20000001 for Point with SRID)
	typ := order.Uint32(data[1:5])
	if typ&0x20000000 == 0 {
		return fmt.Errorf("EWKB type does not have SRID flag: %x", typ)
	}

	// Read SRID
	srid := order.Uint32(data[5:9])
	if srid != 4326 {
		return fmt.Errorf("unexpected SRID: %d (expected 4326)", srid)
	}

	// Read coordinates
	c.Longitude = math.Float64frombits(order.Uint64(data[9:17]))
	c.Latitude = math.Float64frombits(order.Uint64(data[17:25]))

	return nil
} // Value allows Coordinates to be written into Postgres
func (c Coordinates) Value() (driver.Value, error) {
	return fmt.Sprintf("SRID=4326;POINT(%f %f)", c.Longitude, c.Latitude), nil
}

type Venue struct {
	Id                      uuid.UUID              `db:"id" json:"id,omitempty"`
	HostId                  uuid.UUID              `db:"host_id" json:"host_id,omitempty"`
	Images                  []string               `db:"images" json:"images,omitempty"`
	Name                    string                 `db:"name" json:"name,omitempty"`
	VenueType               string                 `db:"venue_type" json:"venue_type,omitempty"`
	Rules                   []string               `db:"rules" json:"rules,omitempty"`
	Accessibility           []string               `db:"accessibility" json:"accessibility,omitempty"`
	MinBookingDurationHours int64                  `db:"min_booking_duration_hours" json:"min_booking_duration_hours,omitempty"`
	CancellationPolicy      string                 `db:"cancellation_policy" json:"cancellation_policy,omitempty"`
	Description             string                 `db:"description" json:"description,omitempty"`
	Location                string                 `db:"location" json:"location,omitempty"`
	Coordinates             Coordinates            `db:"coordinates" json:"coordinates,omitempty"`
	Capacity                int                    `db:"capacity" json:"capacity,omitempty"`
	Amenities               map[string]interface{} `db:"amenities" json:"amenities,omitempty"`
	PricePerHour            float64                `db:"price_per_hour" json:"price_per_hour,omitempty"`
	Availability            map[string]interface{} `db:"availability" json:"availability,omitempty"`
	Status                  VenueStatus            `db:"status" json:"status,omitempty"`
	CreatedAt               time.Time              `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt               time.Time              `db:"updated_at" json:"updated_at,omitempty"`
}
