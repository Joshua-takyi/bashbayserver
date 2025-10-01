package models

import (
	"testing"
)

// Test to verify partial update behavior
func TestPartialUpdateDataProcessing(t *testing.T) {
	// Test that coordinates are properly handled
	updateData := map[string]interface{}{
		"description": "New description",
		"capacity":    200,
		"coordinates": Coordinates{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
	}

	// Simulate the processing logic from UpdateVenue
	processedData := make(map[string]interface{})
	for key, value := range updateData {
		if key == "coordinates" {
			if coords, ok := value.(Coordinates); ok {
				coordsValue, err := coords.Value()
				if err != nil {
					t.Fatalf("Failed to convert coordinates: %v", err)
				}
				processedData[key] = coordsValue
			}
		} else {
			processedData[key] = value
		}
	}

	// Verify that coordinates were processed correctly
	if processedData["coordinates"] == nil {
		t.Error("Coordinates were not processed")
	}

	// Verify other fields remain unchanged
	if processedData["description"] != "New description" {
		t.Error("Description was not preserved")
	}

	if processedData["capacity"] != 200 {
		t.Error("Capacity was not preserved")
	}
}
