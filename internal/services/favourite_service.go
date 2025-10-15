package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/models"
)

type FavouriteService struct {
	favouritesRepo models.FavouriteRepo
}

func NewFavouriteService(favouritesRepo models.FavouriteRepo) *FavouriteService {
	return &FavouriteService{
		favouritesRepo: favouritesRepo,
	}
}

func (fs *FavouriteService) AddToFavourites(ctx context.Context, userId uuid.UUID, itemId string, itemType string) (*models.Favourite, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}
	if strings.TrimSpace(itemId) == "" {
		return nil, fmt.Errorf("item ID cannot be empty")
	}
	if itemType != "venue" && itemType != "event" {
		return nil, fmt.Errorf("item type must be either 'venue' or 'event'")
	}

	return fs.favouritesRepo.AddToFavourites(ctx, userId, itemId, itemType)
}

func (fs *FavouriteService) RemoveFromFavourites(ctx context.Context, userId uuid.UUID, itemId string) error {
	if userId == uuid.Nil {
		return fmt.Errorf("invalid user ID")
	}
	if strings.TrimSpace(itemId) == "" {
		return fmt.Errorf("item ID cannot be empty")
	}

	return fs.favouritesRepo.RemoveFromFavourites(ctx, userId, itemId)
}

func (fs *FavouriteService) GetFavouritesByUserID(ctx context.Context, userId uuid.UUID) ([]*models.Favourite, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("invalid user ID")
	}

	return fs.favouritesRepo.GetFavouritesByUserID(ctx, userId)
}

// func (fs *FavouriteService) GetFavById(ctx context.Context, id primitive.ObjectID) (*models.Favourite, error) {
// 	if id.IsZero() {
// 		return nil, fmt.Errorf("invalid favourite ID")
// 	}

// 	return fs.favouritesRepo.GetFavById(ctx, id)

// }
