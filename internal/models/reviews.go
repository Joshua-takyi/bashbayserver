package models

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VenueReview struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`

	// Core Relationships
	UserID    uuid.UUID `bson:"user_id" json:"user_id"`
	VenueID   uuid.UUID `bson:"venue_id" json:"venue_id"`
	BookingID uuid.UUID `bson:"booking_id" json:"booking_id"`

	// Review Content & Rating
	Rating  int      `bson:"rating" json:"rating" validate:"required,min=1,max=5"`
	Title   string   `bson:"title" json:"title"`
	Comment string   `bson:"comment" json:"comment"`
	Images  []string `bson:"images" json:"images"`

	// Event-Specific Context (Crucial for Event Venues)
	EventID    uuid.UUID `bson:"event_id" json:"event_id,omitempty"` // NEW: Optional link if reviews are tied to a user-created 'Event' document
	EventType  string    `bson:"event_type" json:"event_type"`       // NEW: e.g., "Wedding," "Corporate Party," "Photo Shoot"
	GuestCount int       `bson:"guest_count" json:"guest_count"`     // NEW: Number of attendees (Context for capacity)
	EventDate  time.Time `bson:"event_date" json:"event_date"`       // NEW: Date the event actually occurred (for sorting/filtering)

	// Reviewer Details (Context)
	UserType string `bson:"user_type" json:"user_type"` // NEW: e.g., "Organizer," "Attendee," "Host" (if host leaves feedback)

	// Aggregated/Structured Feedback (For Filtering and Quick Stats)
	LikedFeatures []string `bson:"liked_features" json:"liked_features"` // NEW: Structured feedback (e.g., ["Location", "Cleanliness", "AV Equipment"])

	// Status & Timestamps
	Status    string    `bson:"status" json:"status"` // NEW: e.g., "Pending Approval," "Approved," "Flagged"
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
