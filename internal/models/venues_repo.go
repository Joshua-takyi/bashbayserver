package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type VenuesRepo interface {
	CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID, accessToken string) (*Venue, error)
	ListVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error)
	ListVenuesByHost(ctx context.Context, hostId uuid.UUID, offset, limit int, accessToken string) ([]*Venue, int, error)
	ListVenues(ctx context.Context, offset, limit int) ([]*Venue, int, error)
	UpdateVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, venue map[string]interface{}, accessToken string) (*Venue, error)
	DeleteVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, accessToken string) error
}

func convertRawToVenue(rawVenue map[string]interface{}) (*Venue, error) {
	// Extract and handle coordinates separately
	var coordStr string
	if coords, exists := rawVenue["coordinates"]; exists {
		if str, ok := coords.(string); ok {
			coordStr = str
		}
		delete(rawVenue, "coordinates") // Remove coordinates for clean unmarshaling
	}

	// Convert raw data to venue struct
	venueBytes, err := json.Marshal(rawVenue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw venue: %v", err)
	}

	venue := &Venue{}
	if err := json.Unmarshal(venueBytes, venue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to venue struct: %v", err)
	}

	// Parse coordinates back to struct
	if coordStr != "" {
		if err := venue.Coordinates.Scan([]byte(coordStr)); err != nil {
			return nil, fmt.Errorf("failed to parse coordinates: %v", err)
		}
	}

	return venue, nil
}

// Helper function to convert Venue struct to map for database insertion
func venueToInsertMap(venue *Venue) (map[string]interface{}, error) {
	coordsValue, err := venue.Coordinates.Value()
	if err != nil {
		return nil, fmt.Errorf("failed to convert coordinates: %v", err)
	}

	return map[string]interface{}{
		"id":                         venue.Id,
		"host_id":                    venue.HostId,
		"name":                       venue.Name,
		"images":                     venue.Images,
		"rules":                      venue.Rules,
		"accessibility":              venue.Accessibility,
		"venue_type":                 venue.VenueType,
		"min_booking_duration_hours": venue.MinBookingDurationHours,
		"cancellation_policy":        venue.CancellationPolicy,
		"description":                venue.Description,
		"location":                   venue.Location,
		"coordinates":                coordsValue,
		"capacity":                   venue.Capacity,
		"amenities":                  venue.Amenities,
		"price_per_hour":             venue.PricePerHour,
		"availability":               venue.Availability,
		"status":                     venue.Status,
		"created_at":                 venue.CreatedAt,
		"updated_at":                 venue.UpdatedAt,
	}, nil
}

func (su *SupabaseRepo) CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID, accessToken string) (*Venue, error) {
	// Convert venue to map for database insertion
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err == nil && authClient != nil {
			client = authClient
		}
	}
	venueData, err := venueToInsertMap(venue)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare venue data: %v", err)
	}

	// Insert the new venue into the "venues" table
	data, count, err := client.
		From(VenuesTable).
		Insert(venueData, false, "", "", "exact").
		Execute()

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, fmt.Errorf("no venue was created")
	}

	// Unmarshal response to raw venue data
	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(rawVenues) == 0 {
		return nil, fmt.Errorf("no venue returned from database")
	}

	// Use helper function to convert raw venue to Venue struct
	result, err := convertRawToVenue(rawVenues[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert venue: %v", err)
	}

	return result, nil
}

func (su *SupabaseRepo) GetVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error) {
	return nil, nil
}

func (su *SupabaseRepo) ListVenues(ctx context.Context, offset, limit int) ([]*Venue, int, error) {
	// Get total count
	_, total, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Limit(0, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues count: %v", err)
	}

	data, count, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues: %v", err)
	}

	if count == 0 {
		return []*Venue{}, int(total), nil
	}

	// Unmarshal directly to a slice of maps to handle coordinates specially
	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert venue data: %v", err)
		}
		venues = append(venues, venue)
	}

	return venues, int(total), nil
}

func (su *SupabaseRepo) ListVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error) {
	data, count, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Eq("id", id.String()).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get venue: %v", err)
	}

	if count == 0 {
		return nil, fmt.Errorf("venue not found")
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal venue: %v", err)
	}

	if len(rawVenues) == 0 {
		return nil, fmt.Errorf("venue not found")
	}

	result, err := convertRawToVenue(rawVenues[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert venue: %v", err)
	}

	return result, nil
}

func (su *SupabaseRepo) ListVenuesByHost(ctx context.Context, hostId uuid.UUID, offset, limit int, accessToken string) ([]*Venue, int, error) {
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err == nil && authClient != nil {
			client = authClient
		}
	}
	// Get total count for the host
	_, total, err := client.From(VenuesTable).Select("*", "exact", false).Limit(0, "").Eq("host_id", hostId.String()).Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues count for host: %v", err)
	}

	data, count, err := client.From(VenuesTable).Select("id, host_id, name, description, images, rules, accessibility, venue_type, min_booking_duration_hours, cancellation_policy,location,capacity, amenities, price_per_hour, availability, status, coordinates, created_at, updated_at", "exact", false).Eq("host_id", hostId.String()).Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues: %v", err)
	}

	if count == 0 {
		return []*Venue{}, int(total), nil
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	if len(rawVenues) == 0 {
		return nil, 0, fmt.Errorf("no venues found %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to convert venue %v", err)
		}
		venues = append(venues, venue)
	}

	return venues, int(total), nil
}

func (su *SupabaseRepo) UpdateVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, venue map[string]interface{}, accessToken string) (*Venue, error) {
	if len(venue) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	// Use an authenticated client if an access token was provided by the caller
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err == nil && authClient != nil {
			client = authClient
		}
	}

	// Process the update data to handle coordinates if present
	updateData := make(map[string]interface{})
	for key, value := range venue {
		if key == "coordinates" {
			// Handle coordinates field - convert to proper format if provided
			if coords, ok := value.(*Coordinates); ok {
				coordsValue, err := coords.Value()
				if err != nil {
					return nil, fmt.Errorf("failed to convert coordinates: %v", err)
				}
				updateData[key] = coordsValue
			} else if coords, ok := value.(Coordinates); ok {
				coordsValue, err := coords.Value()
				if err != nil {
					return nil, fmt.Errorf("failed to convert coordinates: %v", err)
				}
				updateData[key] = coordsValue
			} else {
				// If coordinates is provided as raw value, pass it through
				updateData[key] = value
			}
		} else {
			// For all other fields, use the value as-is for partial update
			updateData[key] = value
		}
	}

	data, count, err := client.From(VenuesTable).Update(updateData, "", "exact").Eq("id", venue_id.String()).Eq("host_id", host_id.String()).Execute()

	if err != nil {
		return nil, fmt.Errorf("failed to update venue: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("no venue was updated")
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated venue: %v", err)
	}

	if len(rawVenues) == 0 {
		return nil, fmt.Errorf("no venue returned after update")
	}

	return convertRawToVenue(rawVenues[0])
}

func (su *SupabaseRepo) DeleteVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, accessToken string) error {
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err == nil && authClient != nil {
			client = authClient
		}
	}

	_, count, err := client.From(VenuesTable).Delete("", "exact").Eq("id", venue_id.String()).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete venue: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("no venue was deleted - venue may not exist")
	}

	return nil
}
