package models

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
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
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
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
		// Handle nil coordinates gracefully - set to zero
		c.Latitude = 0
		c.Longitude = 0
		return nil
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

	// Check if it's a hex-encoded EWKB string
	if len(dataStr) >= 32 && isHexString(dataStr) {
		// Decode hex string to bytes
		ewkbBytes, err := hex.DecodeString(dataStr)
		if err != nil {
			return fmt.Errorf("failed to decode EWKB hex: %v", err)
		}

		// Parse EWKB binary format
		return c.parseEWKB(ewkbBytes)
	}

	// If all parsing fails, try to parse as plain coordinates
	// This handles cases where coordinates might be stored as "lat,lng" or similar
	if parts := strings.Split(dataStr, ","); len(parts) == 2 {
		if lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			if lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
				c.Latitude = lat
				c.Longitude = lng
				return nil
			}
		}
	}

	return fmt.Errorf("failed to parse coordinates from: %q", dataStr)
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
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

type DateRange struct {
	Start string `json:"start"` // YYYY-MM-DD
	End   string `json:"end"`   // YYYY-MM-DD; optional, if empty = same as Start
}

type TimeRange struct {
	Start string `json:"start"` // HH:MM (24h)
	End   string `json:"end"`   // HH:MM (24h)
}

type Availability struct {
	// Host-defined blocks (days the venue is NOT available)
	UnavailableDates      []string    `json:"unavailable_dates,omitempty"`       // e.g., ["2025-10-12","2025-10-18"]
	UnavailableDateRanges []DateRange `json:"unavailable_date_ranges,omitempty"` // e.g., [{"start":"2025-12-24","end":"2025-12-26"}]

	// Optional: recurring open hours per weekday ("Mon".."Sun"); useful if you later want hourly booking search
	WeeklyHours map[string][]TimeRange `json:"weekly_hours,omitempty"` // {"Mon":[{"start":"09:00","end":"17:00"}]}

	Timezone string `json:"timezone,omitempty"` // e.g., "America/Los_Angeles"
}

// IsDateUnavailable returns true if a given date (in venue timezone if you later add TZ handling) is blocked.
func (a Availability) IsDateUnavailable(d time.Time) bool {
	ds := d.Format("2006-01-02")
	for _, s := range a.UnavailableDates {
		if s == ds {
			return true
		}
	}

	for _, r := range a.UnavailableDateRanges {
		if r.Start == "" {
			continue
		}
		end := r.End
		if end == "" {
			end = r.Start
		}
		startT, err1 := time.Parse("2006-01-02", r.Start)
		endT, err2 := time.Parse("2006-01-02", end)
		if err1 != nil || err2 != nil {
			continue
		}
		// Inclusive range
		if !d.Before(startT) && !d.After(endT) {
			return true
		}
	}
	return false
}

func daysIn(month time.Month, year int) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// CalendarSnapshot returns a map[dayOfMonth]unavailable for the requested month.
// Use this to paint the calendar.
func (a Availability) CalendarSnapshot(year int, month time.Month) map[int]bool {
	days := daysIn(month, year)
	out := make(map[int]bool, days)
	for day := 1; day <= days; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		out[day] = a.IsDateUnavailable(date)
	}
	return out
}

type Venue struct {
	Id     uuid.UUID `db:"id" json:"id,omitempty"`
	HostId uuid.UUID `db:"host_id" json:"host_id,omitempty"`

	// MARKETING & CORE INFO
	Name         string   `db:"name" json:"name,omitempty"`
	VibeHeadline string   `db:"vibe_headline" json:"vibe_headline,omitempty"`
	Description  string   `db:"description" json:"description,omitempty"`
	Images       []string `db:"images" json:"images,omitempty"`
	VenueType    []string `db:"venue_type" json:"venue_type,omitempty"`
	Slug         string   `db:"slug" json:"slug,omitempty"`
	Tags         []string `db:"tags" json:"tags,omitempty"`
	Region       string   `db:"region" json:"region,omitempty" validate:"required"`
	// CAPACITY & DIMENSIONS
	Capacity          int `db:"capacity" json:"capacity,omitempty"`                       // Total max capacity
	SeatingCapacity   int `db:"seating_capacity" json:"seating_capacity,omitempty"`       // NEW
	StandingCapacity  int `db:"standing_capacity" json:"standing_capacity,omitempty"`     // NEW
	CeilingHeightFeet int `db:"ceiling_height_feet" json:"ceiling_height_feet,omitempty"` // NEW

	// LOCATION & LOGISTICS
	Location      string      `db:"location" json:"location,omitempty"`
	Coordinates   Coordinates `db:"coordinates" json:"coordinates"`
	Accessibility []string    `db:"accessibility" json:"accessibility,omitempty"`
	LoadInAccess  string      `db:"load_in_access" json:"load_in_access,omitempty"`

	// AMENITIES & RULES
	Amenities               map[string]any `db:"amenities" json:"amenities,omitempty"`
	Rules                   []string       `db:"rules" json:"rules,omitempty"`
	AlcoholPolicy           string         `db:"alcohol_policy" json:"alcohol_policy,omitempty"`                       // NEW
	ExternalCateringAllowed bool           `db:"external_catering_allowed" json:"external_catering_allowed,omitempty"` // NEW

	// PRICING & BOOKING
	PriceModel              string  `db:"price_model" json:"price_model,omitempty" validate:"required,oneof=HOURLY FIXED QUOTE_ONLY"` // "HOURLY", "FIXED", "QUOTE_ONLY"
	PricePerHour            float64 `db:"price_per_hour" json:"price_per_hour,omitempty"`
	MinBookingDurationHours int64   `db:"min_booking_duration_hours" json:"min_booking_duration_hours,omitempty"`
	FixedPricePackagePrice  float64 `db:"fixed_price_package_price" json:"fixed_price_package_price,omitempty"`
	PackageDurationHours    int64   `db:"package_duration_hours" json:"package_duration_hours,omitempty"`
	OverTimeRatePerHour     float64 `db:"overtime_rate_per_hour" json:"overtime_rate_per_hour,omitempty"`
	CleaningFee             float64 `db:"cleaning_fee" json:"cleaning_fee,omitempty"`
	SecurityDeposit         float64 `db:"security_deposit" json:"security_deposit,omitempty"`
	// TaxRate                 float64  `db:"tax_rate" json:"tax_rate,omitempty"`
	SetupTakedownDuration float64  `db:"setup_takedown_duration" json:"setup_takedown_duration,omitempty"`
	IncludedItems         []string `db:"included_items" json:"included_items,omitempty"`

	// STATUS & ADMIN
	CancellationPolicy string       `db:"cancellation_policy" json:"cancellation_policy,omitempty"`
	Availability       Availability `db:"availability" json:"availability,omitempty"`
	Status             VenueStatus  `db:"status" json:"status,omitempty"`
	CreatedAt          time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time    `db:"updated_at" json:"updated_at"`
}
