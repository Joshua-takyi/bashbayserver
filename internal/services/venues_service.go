package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/connect"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/models"
)

type VenuesService struct {
	venuesRepo models.VenuesRepo
}

func NewVenuesService(venuesRepo models.VenuesRepo) *VenuesService {
	return &VenuesService{
		venuesRepo: venuesRepo,
	}
}

type ctxKey string

func (vs *VenuesService) CreateVenue(ctx context.Context, venue *models.Venue, hostId uuid.UUID, accessToken string) (*models.Venue, error) {
	if err := models.Validate.Struct(venue); err != nil {
		return nil, fmt.Errorf("invalid venue data provided: %v", err)
	}
	now := time.Now()
	if venue.Id == uuid.Nil {
		venue.Id = uuid.New()
	}

	if accessToken != "" {
		ctx = context.WithValue(ctx, ctxKey("access_token"), accessToken)
	}

	// Upload images first if any
	var uploadedPublicIDs []string
	if len(venue.Images) > 0 {
		// Upload images with timeout
		uploadChan := make(chan struct {
			urls      []string
			publicIDs []string
		}, 1)
		errorChan := make(chan error, 1)

		go func() {
			urls, publicIDs, uploadErr := helpers.UploadImages(ctx, connect.Cld, venue.Images, helpers.VenueFolder)
			if uploadErr != nil {
				errorChan <- uploadErr
				return
			}
			uploadChan <- struct {
				urls      []string
				publicIDs []string
			}{urls, publicIDs}
		}()

		// Wait for upload with timeout
		select {
		case result := <-uploadChan:
			venue.Images = result.urls
			uploadedPublicIDs = result.publicIDs
			fmt.Printf("Successfully uploaded %d images\n", len(result.urls))
		case uploadErr := <-errorChan:
			return nil, fmt.Errorf("failed to upload images: %v", uploadErr)
		case <-time.After(10 * time.Second):
			return nil, fmt.Errorf("image upload timeout")
		}
	}

	venue.HostId = hostId
	venue.CreatedAt = now
	venue.UpdatedAt = now
	venue.Status = models.StatusPending

	// Create the venue in the database with the uploaded image URLs
	createdVenue, err := vs.venuesRepo.CreateVenue(ctx, venue, hostId, accessToken)
	if err != nil {
		// If venue creation fails, clean up uploaded images
		if len(uploadedPublicIDs) > 0 {
			helpers.DeleteImages(ctx, connect.Cld, uploadedPublicIDs)
		}
		return nil, err
	}

	return createdVenue, nil
}

func (vs *VenuesService) ListVenues(ctx context.Context, offset, limit int) ([]*models.Venue, int, error) {

	// Validate input parameters
	if offset < 0 || limit <= 0 {
		return nil, 0, fmt.Errorf("invalid offset or limit")
	}

	return vs.venuesRepo.ListVenues(ctx, offset, limit)
}

func (vs *VenuesService) ListVenueByID(ctx context.Context, id uuid.UUID) (*models.Venue, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("invalid venue ID")
	}

	return vs.venuesRepo.ListVenueByID(ctx, id)
}

func (vs *VenuesService) DeleteVenue(ctx context.Context, host_id uuid.UUID, venue_id uuid.UUID, accessToken string) error {
	if host_id == uuid.Nil || venue_id == uuid.Nil {
		return fmt.Errorf("invalid host ID or venue ID")
	}

	// Store access token in context so repo can create an authenticated client
	if accessToken != "" {
		ctx = context.WithValue(ctx, ctxKey("access_token"), accessToken)
	}

	return vs.venuesRepo.DeleteVenue(ctx, host_id, venue_id, accessToken)
}

func (vs *VenuesService) ListVenuesByHost(ctx context.Context, hostId uuid.UUID, offset, limit int, accessToken string) ([]*models.Venue, int, error) {

	if offset < 0 || limit <= 0 {
		return nil, 0, fmt.Errorf("invalid offset or limit")
	}

	if hostId == uuid.Nil {
		return nil, 0, fmt.Errorf("invalid host ID")
	}

	return vs.venuesRepo.ListVenuesByHost(ctx, hostId, offset, limit, accessToken)
}

func (vs *VenuesService) QueryVenues(ctx context.Context, query map[string]interface{}, offset, limit int) ([]*models.Venue, int, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, fmt.Errorf("invalid offset or limit")
	}
	if len(query) == 0 {
		return nil, 0, fmt.Errorf("query parameters cannot be empty")
	}
	return vs.venuesRepo.QueryVenues(ctx, query, offset, limit)
}
