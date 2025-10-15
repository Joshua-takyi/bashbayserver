package models

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	ReviewStatusPending  = "Pending Approval"
	ReviewStatusApproved = "Approved"
	ReviewStatusFlagged  = "Flagged"
	ReviewDbName         = "bashbay"
	ReviewColName        = "venue_reviews"
)

type ReviewsRepo interface {
	CreateReview(ctx context.Context, userId uuid.UUID, venueId uuid.UUID, review *VenueReview) (*VenueReview, error)
	GetReviewsByVenue(ctx context.Context, venueId uuid.UUID) ([]*VenueReview, error)
	GetReviewsByUser(ctx context.Context, userId uuid.UUID) ([]*VenueReview, error)
	UpdateReview(ctx context.Context, userId uuid.UUID, reviewId uuid.UUID, updatedReview *VenueReview) (*VenueReview, error)
	DeleteReview(ctx context.Context, userId uuid.UUID, reviewId uuid.UUID) error
}

func (r *VenueReview) BeforeCreate() error {
	if r.ID.IsZero() {
		r.ID = primitive.NewObjectID()
	}
	return nil
}

func (r VenueReview) ValidateReview() error {

	if r.Rating < 1 || r.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	if r.UserID == uuid.Nil {
		return fmt.Errorf("invalid user ID")
	}

	if r.VenueID == uuid.Nil {
		return fmt.Errorf("invalid venue ID")
	}

	return nil
}

func (r *VenueReview) Sanitize() {
	r.Title = helpers.StringTrim(r.Title)
	r.Comment = helpers.StringTrim(r.Comment)

	// Ensure rating is within bounds
	if r.Rating < 1 {
		r.Rating = 1
	} else if r.Rating > 5 {
		r.Rating = 5
	}
	// Remove duplicate liked features
	r.LikedFeatures = helpers.RemoveDuplicates(r.LikedFeatures)
	// Remove profanity from comment
	clean_comment := helpers.RemoveProfanity(r.Comment)
	r.Comment = clean_comment
}

func (mdb *MongodbRepo) GetCollection(ctx context.Context, dbName, colName string) (*mongo.Collection, error) {
	if mdb.mongodbClient == nil {
		return nil, fmt.Errorf("mongodb client is not initialized")
	}
	client := mdb.mongodbClient.Database(dbName).Collection(colName)
	return client, nil
}

func (mdb *MongodbRepo) CreateReview(ctx context.Context, userId uuid.UUID, venueId uuid.UUID, review *VenueReview) (*VenueReview, error) {
	if err := review.ValidateReview(); err != nil {
		return nil, fmt.Errorf("invalid review data: %w", err)
	}

	if err := review.BeforeCreate(); err != nil {
		return nil, fmt.Errorf("failed to prepare review for creation: %w", err)
	}
	col, err := mdb.GetCollection(ctx, ReviewDbName, ReviewColName)

	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}
	_, err = col.InsertOne(ctx, review)
	if err != nil {
		return nil, fmt.Errorf("failed to insert review into database: %w", err)
	}

	return review, nil
}

func (mdb *MongodbRepo) GetReviewsByVenue(ctx context.Context, venueId uuid.UUID) ([]*VenueReview, error) {
	return nil, nil
}
func (mdb *MongodbRepo) GetReviewsByUser(ctx context.Context, userId uuid.UUID) ([]*VenueReview, error) {
	return nil, nil
}
func (mdb *MongodbRepo) UpdateReview(ctx context.Context, userId uuid.UUID, reviewId uuid.UUID, updatedReview *VenueReview) (*VenueReview, error) {
	return nil, nil
}
func (mdb *MongodbRepo) DeleteReview(ctx context.Context, userId uuid.UUID, reviewId uuid.UUID) error {
	return nil
}
