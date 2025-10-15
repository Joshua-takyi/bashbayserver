package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	FavouriteDbName  = "bashbay"
	FavouriteColName = "favourites"
)

type FavouriteItem struct {
	ItemID   string    `bson:"item_id" json:"item_id"`
	ItemType string    `bson:"item_type" json:"item_type"` // "venue" or "event"
	AddedAt  time.Time `bson:"added_at" json:"added_at"`
}

type Favourite struct {
	ID        primitive.ObjectID       `bson:"_id,omitempty" json:"id"`
	UserID    uuid.UUID                `bson:"user_id" json:"user_id" validate:"required"`
	Items     map[string]FavouriteItem `bson:"items" json:"items"`
	CreatedAt time.Time                `bson:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt time.Time                `bson:"updated_at,omitempty" json:"updated_at,omitempty"`
}

type FavouriteRepo interface {
	AddToFavourites(ctx context.Context, userId uuid.UUID, itemId string, itemType string) (*Favourite, error)
	RemoveFromFavourites(ctx context.Context, userId uuid.UUID, itemId string) error
	// GetFavById(ctx context.Context, id primitive.ObjectID) (*Favourite, error)
	GetFavouritesByUserID(ctx context.Context, userId uuid.UUID) ([]*Favourite, error)
}

func (f *Favourite) BeforeCreate() error {
	if f.ID.IsZero() {
		f.ID = primitive.NewObjectID()
	}
	return nil
}

func (mdb *MongodbRepo) AddToFavourites(ctx context.Context, userId uuid.UUID, itemId string, itemType string) (*Favourite, error) {
	col, err := mdb.GetCollection(ctx, FavouriteDbName, FavouriteColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}
	now := time.Now()
	filter := bson.M{"user_id": userId}

	update := bson.M{
		"$set": bson.M{
			"updated_at": now,
			fmt.Sprintf("items.%s", itemId): FavouriteItem{
				ItemID:   itemId,
				ItemType: itemType,
				AddedAt:  now,
			},
		},
		"$setOnInsert": bson.M{
			"user_id":    userId,
			"created_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result Favourite
	err = col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error upserting favourite: %v", err)
	}

	return &result, nil
}

func (mdb *MongodbRepo) RemoveFromFavourites(ctx context.Context, userId uuid.UUID, itemId string) error {
	col, err := mdb.GetCollection(ctx, FavouriteDbName, FavouriteColName)
	if err != nil {
		return fmt.Errorf("error getting collection: %v", err)
	}

	filter := bson.M{"user_id": userId}
	update := bson.M{
		"$unset": bson.M{
			fmt.Sprintf("items.%s", itemId): "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	_, err = col.UpdateOne(ctx, filter, update)
	return err
}

func (mdb *MongodbRepo) GetFavouritesByUserID(ctx context.Context, userId uuid.UUID) ([]*Favourite, error) {
	col, err := mdb.GetCollection(ctx, FavouriteDbName, FavouriteColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}

	filter := bson.M{"user_id": userId}
	cursor, err := col.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error finding favourites: %v", err)
	}
	defer cursor.Close(ctx)

	var favourites []*Favourite
	for cursor.Next(ctx) {
		var fav Favourite
		if err := cursor.Decode(&fav); err != nil {
			return nil, fmt.Errorf("error decoding favourite: %v", err)
		}
		favourites = append(favourites, &fav)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return favourites, nil
}

// func (mdb *MongodbRepo) GetFavById(ctx context.Context, id primitive.ObjectID) (*Favourite, error) {
// 	col, err := mdb.GetCollection(ctx, FavouriteDbName, FavouriteColName)
// 	if err != nil {
// 		return nil, fmt.Errorf("error getting collection: %v", err)
// 	}

// 	var fav Favourite
// 	err = col.FindOne(ctx, bson.M{"_id": id}).Decode(&fav)
// 	if err != nil {
// 		return nil, fmt.Errorf("error finding favourite by ID: %v", err)
// 	}

// 	return &fav, nil
// }
