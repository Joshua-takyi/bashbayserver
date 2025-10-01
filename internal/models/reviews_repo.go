package models

import (
	"context"

	"github.com/google/uuid"
)

type ReviewRepository interface {
	CreateReview(ctx context.Context, userId uuid.UUID, review *Review) error
}
