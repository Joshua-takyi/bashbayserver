package models

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID        uuid.UUID `bson:"id" json:"id"`
	UserID    uuid.UUID `bson:"user_id" json:"user_id"`
	ProductID uuid.UUID `bson:"product_id" json:"product_id"`
	Rating    int       `bson:"rating" json:"rating" validate:"required,min=1,max=5"`
	Comment   string    `bson:"comment" json:"comment"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}
