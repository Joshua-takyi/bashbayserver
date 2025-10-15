package models

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type VenuesRepo interface {
	CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID, accessToken string) (*Venue, error)
	ListVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error)
	ListVenuesByHost(ctx context.Context, hostId uuid.UUID, offset, limit int, accessToken string) ([]*Venue, int, error)
	ListVenues(ctx context.Context, offset, limit int) ([]*Venue, int, error)
	UpdateVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, venue map[string]interface{}, accessToken string) (*Venue, error)
	DeleteVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, accessToken string) error
	QueryVenues(ctx context.Context, query map[string]interface{}, offset, limit int) ([]*Venue, int, error)
	GetVenueBySlug(ctx context.Context, slug string) (*Venue, error)
	CreateManyVenues(ctx context.Context, venues []*Venue, hostId uuid.UUID, accessToken string) ([]*Venue, error)
}

func (su *SupabaseRepo) getClientWithAuth(accessToken string) *supabase.Client {
	if accessToken != "" {
		if authClient, err := su.GetAuthenticatedClient(accessToken); err == nil && authClient != nil {
			return authClient
		}
	}
	return su.supabaseClient
}

func convertRawToVenue(rawVenue map[string]interface{}) (*Venue, error) {
	var coordStr string
	if coords, exists := rawVenue["coordinates"]; exists {
		if str, ok := coords.(string); ok {
			coordStr = str
		}
		delete(rawVenue, "coordinates")
	}

	// Handle array fields that might come as strings from the database
	arrayFields := []string{"images", "rules", "accessibility", "tags", "included_items", "venue_type", "amenities", "availability"} // Add other array fields as needed
	for _, field := range arrayFields {
		if val, exists := rawVenue[field]; exists && val != nil {
			// If it's already a slice, keep it
			if _, ok := val.([]interface{}); ok {
				continue
			}
			// If it's a string, try to parse it as JSON
			if str, ok := val.(string); ok {
				var arr []string
				if err := json.Unmarshal([]byte(str), &arr); err != nil {
					// If parsing fails, treat as empty array
					rawVenue[field] = []string{}
				} else {
					rawVenue[field] = arr
				}
			}
		}
	}
	// Availability might come as JSON string; normalize it to an object
	if val, exists := rawVenue["availability"]; exists && val != nil {
		switch v := val.(type) {
		case string:
			var a Availability
			if err := json.Unmarshal([]byte(v), &a); err == nil {
				rawVenue["availability"] = a
			}
		}
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
		"vibe_headline":              venue.VibeHeadline,
		"min_booking_duration_hours": venue.MinBookingDurationHours,
		"region":                     venue.Region,
		"cancellation_policy":        venue.CancellationPolicy,
		"description":                venue.Description,
		"location":                   venue.Location,
		"coordinates":                coordsValue,
		"capacity":                   venue.Capacity,
		"amenities":                  venue.Amenities,
		"price_per_hour":             venue.PricePerHour,
		"price_model":                venue.PriceModel,
		"seating_capacity":           venue.SeatingCapacity,
		"standing_capacity":          venue.StandingCapacity,
		"ceiling_height_feet":        venue.CeilingHeightFeet,
		"tags":                       venue.Tags,
		"alcohol_policy":             venue.AlcoholPolicy,
		"external_catering_allowed":  venue.ExternalCateringAllowed,
		"cleaning_fee":               venue.CleaningFee,
		"security_deposit":           venue.SecurityDeposit,
		"setup_takedown_duration":    venue.SetupTakedownDuration,
		"included_items":             venue.IncludedItems,
		"slug":                       venue.Slug,
		"overtime_rate_per_hour":     venue.OverTimeRatePerHour,
		"fixed_price_package_price":  venue.FixedPricePackagePrice,
		"package_duration_hours":     venue.PackageDurationHours,
		"load_in_access":             venue.LoadInAccess,
		"availability":               venue.Availability,
		"status":                     venue.Status,
		"created_at":                 venue.CreatedAt,
		"updated_at":                 venue.UpdatedAt,
	}, nil
}

func (su *SupabaseRepo) CreateVenue(ctx context.Context, venue *Venue, hostId uuid.UUID, accessToken string) (*Venue, error) {
	venueData, err := venueToInsertMap(venue)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare venue data: %v", err)
	}

	client := su.getClientWithAuth(accessToken)
	data, count, err := client.From(VenuesTable).Insert(venueData, false, "", "", "exact").Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create venue: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("no venue was created")
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	if len(rawVenues) == 0 {
		return nil, fmt.Errorf("no venue returned from database")
	}

	return convertRawToVenue(rawVenues[0])
}

func (su *SupabaseRepo) GetVenueByID(ctx context.Context, id uuid.UUID) (*Venue, error) {
	return nil, nil
}

func (su *SupabaseRepo) ListVenues(ctx context.Context, offset, limit int) ([]*Venue, int, error) {
	_, total, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Limit(0, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues count: %v", err)
	}

	data, _, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues: %v", err)
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, 0, err
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

	return convertRawToVenue(rawVenues[0])
}

func (su *SupabaseRepo) ListVenuesByHost(ctx context.Context, hostId uuid.UUID, offset, limit int, accessToken string) ([]*Venue, int, error) {
	client := su.getClientWithAuth(accessToken)

	_, total, err := client.From(VenuesTable).Select("*", "exact", false).Eq("host_id", hostId.String()).Limit(0, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues count: %v", err)
	}

	data, _, err := client.From(VenuesTable).Select("*", "exact", false).Eq("host_id", hostId.String()).Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues: %v", err)
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, 0, err
		}
		venues = append(venues, venue)
	}

	return venues, int(total), nil
}

func (su *SupabaseRepo) UpdateVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, venue map[string]interface{}, accessToken string) (*Venue, error) {
	if len(venue) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	client := su.getClientWithAuth(accessToken)

	updateData := make(map[string]interface{})
	for key, value := range venue {
		if key == "coordinates" {
			if coords, ok := value.(*Coordinates); ok {
				if coordsValue, err := coords.Value(); err == nil {
					updateData[key] = coordsValue
				} else {
					return nil, fmt.Errorf("failed to convert coordinates: %v", err)
				}
			} else if coords, ok := value.(Coordinates); ok {
				if coordsValue, err := coords.Value(); err == nil {
					updateData[key] = coordsValue
				} else {
					return nil, fmt.Errorf("failed to convert coordinates: %v", err)
				}
			} else {
				updateData[key] = value
			}
		} else {
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
	client := su.getClientWithAuth(accessToken)

	_, count, err := client.From(VenuesTable).Delete("", "exact").Eq("id", venue_id.String()).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete venue: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("no venue was deleted")
	}

	return nil
}

func (su *SupabaseRepo) QueryVenues(ctx context.Context, query map[string]interface{}, offset, limit int) ([]*Venue, int, error) {
	if len(query) == 0 {
		return su.ListVenues(ctx, offset, limit)
	}

	// Build the base query for counting - select only id to avoid geometry parsing issues
	// Using "*" with exact=true returns count in header, but PostgREST still tries to parse geometry
	// So we select a simple column instead
	countQuery := su.supabaseClient.From(VenuesTable).Select("id", "exact", false)

	// Apply filters to count query
	for key, value := range query {
		switch key {
		case "venue_type":
			// venue_type is an array column
			// Since PostgREST array operators are causing issues, use ILIKE as fallback
			// This will match if the stringified array contains the search term
			switch v := value.(type) {
			case []string:
				if len(v) > 0 {
					// Match any of the venue types
					for _, vt := range v {
						countQuery = countQuery.Ilike("venue_type", "%"+vt+"%")
					}
				}
			case []interface{}:
				vals := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok && s != "" {
						vals = append(vals, s)
					}
				}
				if len(vals) > 0 {
					for _, vt := range vals {
						countQuery = countQuery.Ilike("venue_type", "%"+vt+"%")
					}
				}
			case string:
				if v != "" {
					// Single string value
					countQuery = countQuery.Ilike("venue_type", "%"+v+"%")
				}
			}
		case "min_price":
			if minPrice, ok := value.(float64); ok {
				countQuery = countQuery.Gte("price_per_hour", fmt.Sprintf("%f", minPrice))
			}
		case "max_price":
			if maxPrice, ok := value.(float64); ok {
				countQuery = countQuery.Lte("price_per_hour", fmt.Sprintf("%f", maxPrice))
			}
		case "min_capacity":
			if minCap, ok := value.(int); ok {
				countQuery = countQuery.Gte("capacity", fmt.Sprintf("%d", minCap))
			}
		case "max_capacity":
			if maxCap, ok := value.(int); ok {
				countQuery = countQuery.Lte("capacity", fmt.Sprintf("%d", maxCap))
			}
		case "location":
			if location, ok := value.(string); ok && location != "" {
				countQuery = countQuery.Ilike("location", "%"+location+"%")
			}
		case "status":
			if status, ok := value.(string); ok && status != "" {
				countQuery = countQuery.Ilike("status", status)
			}
		case "name":
			if name, ok := value.(string); ok && name != "" {
				countQuery = countQuery.Ilike("name", "%"+name+"%")
			}
		case "description":
			if desc, ok := value.(string); ok && desc != "" {
				countQuery = countQuery.Ilike("description", "%"+desc+"%")
			}
		case "amenities":
			if amenities, ok := value.([]string); ok && len(amenities) > 0 {
				for _, amenity := range amenities {
					countQuery = countQuery.Ilike("amenities", "%"+amenity+"%")
				}
			}
		case "region":
			if region, ok := value.(string); ok && region != "" {
				countQuery = countQuery.Ilike("region", region)
			}
		}
	}

	// Execute count query - use head request to avoid parsing issues
	_, total, err := countQuery.Limit(1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get venues count: %v", err)
	}

	// Build main query with same filters - select all columns
	mainQuery := su.supabaseClient.From(VenuesTable).Select("*", "exact", false)

	for key, value := range query {
		switch key {
		case "venue_type":
			// venue_type is an array column - use ILIKE for compatibility
			switch v := value.(type) {
			case []string:
				if len(v) > 0 {
					for _, vt := range v {
						mainQuery = mainQuery.Ilike("venue_type", "%"+vt+"%")
					}
				}
			case []interface{}:
				vals := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok && s != "" {
						vals = append(vals, s)
					}
				}
				if len(vals) > 0 {
					for _, vt := range vals {
						mainQuery = mainQuery.Ilike("venue_type", "%"+vt+"%")
					}
				}
			case string:
				if v != "" {
					mainQuery = mainQuery.Ilike("venue_type", "%"+v+"%")
				}
			}
		case "min_price":
			if minPrice, ok := value.(float64); ok {
				mainQuery = mainQuery.Gte("price_per_hour", fmt.Sprintf("%f", minPrice))
			}
		case "max_price":
			if maxPrice, ok := value.(float64); ok {
				mainQuery = mainQuery.Lte("price_per_hour", fmt.Sprintf("%f", maxPrice))
			}
		case "min_capacity":
			if minCap, ok := value.(int); ok {
				mainQuery = mainQuery.Gte("capacity", fmt.Sprintf("%d", minCap))
			}
		case "max_capacity":
			if maxCap, ok := value.(int); ok {
				mainQuery = mainQuery.Lte("capacity", fmt.Sprintf("%d", maxCap))
			}
		case "location":
			if location, ok := value.(string); ok && location != "" {
				mainQuery = mainQuery.Ilike("location", "%"+location+"%")
			}
		case "status":
			if status, ok := value.(string); ok && status != "" {
				mainQuery = mainQuery.Ilike("status", status)
			}
		case "name":
			if name, ok := value.(string); ok && name != "" {
				mainQuery = mainQuery.Ilike("name", "%"+name+"%")
			}
		case "description":
			if desc, ok := value.(string); ok && desc != "" {
				mainQuery = mainQuery.Ilike("description", "%"+desc+"%")
			}
		case "amenities":
			if amenities, ok := value.([]string); ok && len(amenities) > 0 {
				for _, amenity := range amenities {
					mainQuery = mainQuery.Ilike("amenities", "%"+amenity+"%")
				}
			}
		case "region":
			if region, ok := value.(string); ok && region != "" {
				mainQuery = mainQuery.Ilike("region", region)
			}
		}
	}

	// Execute main query with pagination
	data, _, err := mainQuery.Range(offset, offset+limit-1, "").Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query venues: %v", err)
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal venues: %v", err)
	}

	venues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, 0, err
		}
		venues = append(venues, venue)
	}

	return venues, int(total), nil
}

func (su *SupabaseRepo) GetVenueBySlug(ctx context.Context, slug string) (*Venue, error) {
	data, count, err := su.supabaseClient.From(VenuesTable).Select("*", "exact", false).Eq("slug", slug).Execute()
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

	return convertRawToVenue(rawVenues[0])
}

func (su *SupabaseRepo) CreateManyVenues(ctx context.Context, venues []*Venue, hostId uuid.UUID, accessToken string) ([]*Venue, error) {
	if len(venues) == 0 {
		return nil, fmt.Errorf("no venues to create")
	}

	insertData := make([]map[string]interface{}, 0, len(venues))
	for _, venue := range venues {
		venue.HostId = hostId
		data, err := venueToInsertMap(venue)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare venue data: %v", err)
		}
		insertData = append(insertData, data)
	}

	client := su.getClientWithAuth(accessToken)
	data, count, err := client.From(VenuesTable).Insert(insertData, false, "", "", "exact").Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create venues: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("no venues were created")
	}

	var rawVenues []map[string]interface{}
	if err := json.Unmarshal(data, &rawVenues); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	if len(rawVenues) == 0 {
		return nil, fmt.Errorf("no venues returned from database")
	}

	createdVenues := make([]*Venue, 0, len(rawVenues))
	for _, raw := range rawVenues {
		venue, err := convertRawToVenue(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to convert venue: %v", err)
		}
		createdVenues = append(createdVenues, venue)
	}

	return createdVenues, nil
}
