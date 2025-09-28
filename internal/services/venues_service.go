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

func (vs *VenuesService) CreateVenue(ctx context.Context, venue *models.Venue, hostId uuid.UUID) (*models.Venue, error) {
	if err := models.Validate.Struct(venue); err != nil {
		return nil, fmt.Errorf("invalid venue data provided: %v", err)
	}
	now := time.Now()
	if venue.Id == uuid.Nil {
		venue.Id = uuid.New()
	}

	// Store original images for upload after successful venue creation
	originalImages := make([]string, len(venue.Images))
	copy(originalImages, venue.Images)

	venue.HostId = hostId
	venue.CreatedAt = now
	venue.UpdatedAt = now
	venue.Status = models.StatusPending
	venue.Images = []string{} // Clear images temporarily

	// First, create the venue in the database
	createdVenue, err := vs.venuesRepo.CreateVenue(ctx, venue, hostId)
	if err != nil {
		return nil, err // Return early if venue creation fails
	}

	// Only upload images if venue creation was successful
	if len(originalImages) > 0 {
		// Upload images with timeout
		uploadChan := make(chan []string, 1)
		errorChan := make(chan error, 1)

		go func() {
			urls, uploadErr := helpers.UploadImages(ctx, connect.Cld, originalImages, helpers.VenueFolder)
			if uploadErr != nil {
				errorChan <- uploadErr
				return
			}
			uploadChan <- urls
		}()

		// Wait for upload with timeout
		select {
		case urls := <-uploadChan:
			createdVenue.Images = urls
			fmt.Printf("Successfully uploaded %d images\n", len(urls))
		case uploadErr := <-errorChan:
			fmt.Printf("Failed to upload images: %v\n", uploadErr)
			// Images failed to upload, but venue is already created
		case <-time.After(10 * time.Second):
			fmt.Printf("Image upload timeout\n")
			// Timeout occurred, but venue is already created
		}
	}

	return createdVenue, nil
}

func (vs *VenuesService) GetVenueByID(ctx context.Context, id uuid.UUID) (*models.Venue, error) {
	return vs.venuesRepo.GetVenueByID(ctx, id)
}

func (vs *VenuesService) ListVenues(ctx context.Context, offset, limit int) ([]*models.Venue, error) {

	// Validate input parameters
	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("invalid offset or limit")
	}

	return vs.venuesRepo.ListVenues(ctx, offset, limit)
}
