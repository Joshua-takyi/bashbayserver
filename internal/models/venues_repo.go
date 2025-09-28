package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type VenuesRepo interface {
	CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID) (*Venue, error)
	GetVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error)
	ListVenues(ctx context.Context, offset, limit int) ([]*Venue, error)
}

func (su *SupabaseRepo) CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID) (*Venue, error) {
	// Create an intermediate struct for unmarshaling from Supabase
	type VenueResponse struct {
		Id           uuid.UUID              `json:"id"`
		HostId       uuid.UUID              `json:"host_id"`
		Name         string                 `json:"name"`
		Images       []string               `json:"images"`
		Description  string                 `json:"description"`
		Location     string                 `json:"location"`
		Coordinates  string                 `json:"coordinates"` // This comes back as a string from PostGIS
		Capacity     int                    `json:"capacity"`
		Amenities    map[string]interface{} `json:"amenities"`
		PricePerHour float64                `json:"price_per_hour"`
		Availability map[string]interface{} `json:"availability"`
		Status       VenueStatus            `json:"status"`
		CreatedAt    string                 `json:"created_at"` // Supabase returns timestamps as strings
		UpdatedAt    string                 `json:"updated_at"`
	}

	var createdResponse []VenueResponse

	// Convert coordinates to PostGIS format manually for Supabase REST API
	venueData := map[string]interface{}{
		"id":             venue.Id,
		"host_id":        venue.HostId,
		"name":           venue.Name,
		"images":         venue.Images,
		"description":    venue.Description,
		"location":       venue.Location,
		"coordinates":    fmt.Sprintf("SRID=4326;POINT(%f %f)", venue.Coordinates.Longitude, venue.Coordinates.Latitude),
		"capacity":       venue.Capacity,
		"amenities":      venue.Amenities,
		"price_per_hour": venue.PricePerHour,
		"availability":   venue.Availability,
		"status":         venue.Status,
		"created_at":     venue.CreatedAt,
		"updated_at":     venue.UpdatedAt,
	}

	// Insert the new venue into the "venues" table
	data, count, err := su.supabaseClient.
		From(VenuesTable).
		Insert(venueData, false, "", "", "exact").
		Execute()

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &createdResponse); err != nil {
		return nil, err
	}

	if count == 0 || len(createdResponse) == 0 {
		return nil, nil // or return an appropriate error
	}

	// Convert the response back to our Venue struct
	result := &Venue{
		Id:           createdResponse[0].Id,
		HostId:       createdResponse[0].HostId,
		Name:         createdResponse[0].Name,
		Images:       createdResponse[0].Images,
		Description:  createdResponse[0].Description,
		Location:     createdResponse[0].Location,
		Capacity:     createdResponse[0].Capacity,
		Amenities:    createdResponse[0].Amenities,
		PricePerHour: createdResponse[0].PricePerHour,
		Availability: createdResponse[0].Availability,
		Status:       createdResponse[0].Status,
	}

	// Parse the coordinates string back to Coordinates struct
	err = result.Coordinates.Scan([]byte(createdResponse[0].Coordinates))
	if err != nil {
		return nil, fmt.Errorf("failed to parse coordinates: %v", err)
	}

	// Parse timestamps if needed (though we might not need them for the response)
	// For now, let's use the original timestamps from the request
	result.CreatedAt = venue.CreatedAt
	result.UpdatedAt = venue.UpdatedAt

	return result, nil
}

func (su *SupabaseRepo) GetVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error) {
	return nil, nil
}

func (su *SupabaseRepo) ListVenues(ctx context.Context, offset, limit int) ([]*Venue, error) {
	data, count, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get venues: %v", err)
	}

	if count == 0 {
		return []*Venue{}, nil
	}

	// Unmarshal directly to a slice of maps to handle coordinates specially
	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue := &Venue{}

		// Extract and remove coordinates from raw data before unmarshaling
		var coordStr string
		if coords, exists := raw["coordinates"]; exists {
			if str, ok := coords.(string); ok {
				coordStr = str
			}
			delete(raw, "coordinates") // Remove coordinates from raw data
		}

		// Marshal back to JSON and unmarshal to venue struct for other fields
		venueData, _ := json.Marshal(raw)
		if err := json.Unmarshal(venueData, venue); err != nil {
			return nil, fmt.Errorf("failed to convert venue data: %v", err)
		}

		// Handle coordinates separately since it comes as PostGIS string
		if coordStr != "" {
			if err := venue.Coordinates.Scan([]byte(coordStr)); err != nil {
				return nil, fmt.Errorf("failed to parse coordinates for venue %v: %v", raw["id"], err)
			}
		}

		venues = append(venues, venue)
	}

	return venues, nil
}
