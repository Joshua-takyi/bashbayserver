package models

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	VenueViewsDbName  = "bashbay"
	VenueViewsColName = "venue_views"
)

type VenueView struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VenueID   string             `bson:"venue_id" json:"venue_id" validate:"required"`
	HostID    string             `bson:"host_id" json:"host_id" validate:"required"` // Add host_id for efficient queries
	UserID    *string            `bson:"user_id,omitempty" json:"user_id,omitempty"` // Optional, for authenticated users
	SessionID string             `bson:"session_id" json:"session_id" validate:"required"`
	IPAddress string             `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent string             `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	ViewedAt  time.Time          `bson:"viewed_at" json:"viewed_at"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"` // TTL index field
}

type VenueViewStats struct {
	VenueID       string `json:"venue_id"`
	TotalViews    int64  `json:"total_views"`
	UniqueViews   int64  `json:"unique_views"`
	ViewsToday    int64  `json:"views_today"`
	ViewsThisWeek int64  `json:"views_this_week"`
}

// HostViewStats represents aggregated view statistics for a host
type HostViewStats struct {
	HostID        string `json:"host_id"`
	TotalViews    int64  `json:"total_views"`
	UniqueViews   int64  `json:"unique_views"`
	ViewsToday    int64  `json:"views_today"`
	ViewsThisWeek int64  `json:"views_this_week"`
	TotalVenues   int64  `json:"total_venues"`
}

type VenueViewsRepo interface {
	TrackVenueView(ctx context.Context, view *VenueView) error
	GetVenueViewStats(ctx context.Context, venueId string, days int) (*VenueViewStats, error)
	GetVenueViewHistory(ctx context.Context, venueId string, limit int) ([]*VenueView, error)
	GetHostViewStats(ctx context.Context, hostId string, days int) (*HostViewStats, error)
	GetHostViewHistory(ctx context.Context, hostId string, limit int) ([]*VenueView, error)
	EnsureIndexes(ctx context.Context) error
}

// EnsureIndexes creates necessary indexes including TTL
func (mdb *MongodbRepo) EnsureIndexes(ctx context.Context) error {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return fmt.Errorf("error getting collection: %v", err)
	}

	indexes := []mongo.IndexModel{
		// TTL index - documents expire after 30 days
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().
				SetExpireAfterSeconds(0). // Expire at the time specified in expires_at
				SetName("expires_at_ttl"),
		},
		// Compound index for unique session views
		{
			Keys: bson.D{
				{Key: "venue_id", Value: 1},
				{Key: "session_id", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("venue_session_unique"),
		},
		// Index for venue queries
		{
			Keys:    bson.D{{Key: "venue_id", Value: 1}},
			Options: options.Index().SetName("venue_id_idx"),
		},
		// Index for host queries (PERFORMANCE BOOST)
		{
			Keys:    bson.D{{Key: "host_id", Value: 1}},
			Options: options.Index().SetName("host_id_idx"),
		},
		// Compound index for host analytics
		{
			Keys: bson.D{
				{Key: "host_id", Value: 1},
				{Key: "viewed_at", Value: -1},
			},
			Options: options.Index().SetName("host_viewed_at_idx"),
		},
		// Index for date range queries
		{
			Keys: bson.D{
				{Key: "venue_id", Value: 1},
				{Key: "viewed_at", Value: -1},
			},
			Options: options.Index().SetName("venue_viewed_at_idx"),
		},
	}

	_, err = col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("error creating indexes: %v", err)
	}

	return nil
}

// TrackVenueView records a venue view with TTL and rate limiting
func (mdb *MongodbRepo) TrackVenueView(ctx context.Context, view *VenueView) error {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return fmt.Errorf("error getting collection: %v", err)
	}

	// Check if this session has viewed this venue recently (within last hour)
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	var recentView VenueView
	err = col.FindOne(ctx, bson.M{
		"venue_id":   view.VenueID,
		"session_id": view.SessionID,
		"viewed_at":  bson.M{"$gte": oneHourAgo},
	}).Decode(&recentView)

	if err == nil {
		// Already viewed within the last hour, don't track again
		return nil
	}

	// Set timestamps
	now := time.Now()
	view.ViewedAt = now
	view.ExpiresAt = now.Add(30 * 24 * time.Hour) // Expire after 30 days

	// Generate ID if not set
	if view.ID.IsZero() {
		view.ID = primitive.NewObjectID()
	}

	// Try to insert
	_, err = col.InsertOne(ctx, view)
	if err != nil {
		// If duplicate key error (same venue_id + session_id), it's not a new view
		if mongo.IsDuplicateKeyError(err) {
			return nil // Silently ignore duplicate views from same session
		}
		return fmt.Errorf("error inserting venue view: %v", err)
	}

	return nil
}

// GetVenueViewStats returns aggregated view statistics
func (mdb *MongodbRepo) GetVenueViewStats(ctx context.Context, venueId string, days int) (*VenueViewStats, error) {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}

	stats := &VenueViewStats{
		VenueID: venueId,
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfWeek := startOfDay.AddDate(0, 0, -int(now.Weekday()))

	// Get total views
	totalCount, err := col.CountDocuments(ctx, bson.M{"venue_id": venueId})
	if err != nil {
		return nil, fmt.Errorf("error counting total views: %v", err)
	}
	stats.TotalViews = totalCount

	// Get unique views (by session_id)
	uniquePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"venue_id": venueId}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$session_id",
		}}},
		{{Key: "$count", Value: "unique_sessions"}},
	}
	uniqueCursor, err := col.Aggregate(ctx, uniquePipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregating unique views: %v", err)
	}
	defer uniqueCursor.Close(ctx)

	var uniqueResult []bson.M
	if err := uniqueCursor.All(ctx, &uniqueResult); err != nil {
		return nil, fmt.Errorf("error decoding unique views: %v", err)
	}
	if len(uniqueResult) > 0 {
		if count, ok := uniqueResult[0]["unique_sessions"].(int32); ok {
			stats.UniqueViews = int64(count)
		}
	}

	// Get views today
	todayCount, err := col.CountDocuments(ctx, bson.M{
		"venue_id":  venueId,
		"viewed_at": bson.M{"$gte": startOfDay},
	})
	if err != nil {
		return nil, fmt.Errorf("error counting today's views: %v", err)
	}
	stats.ViewsToday = todayCount

	// Get views this week
	weekCount, err := col.CountDocuments(ctx, bson.M{
		"venue_id":  venueId,
		"viewed_at": bson.M{"$gte": startOfWeek},
	})
	if err != nil {
		return nil, fmt.Errorf("error counting this week's views: %v", err)
	}
	stats.ViewsThisWeek = weekCount

	return stats, nil
}

// GetVenueViewHistory returns recent view records
func (mdb *MongodbRepo) GetVenueViewHistory(ctx context.Context, venueId string, limit int) ([]*VenueView, error) {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "viewed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := col.Find(ctx, bson.M{"venue_id": venueId}, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding venue views: %v", err)
	}
	defer cursor.Close(ctx)

	var views []*VenueView
	if err := cursor.All(ctx, &views); err != nil {
		return nil, fmt.Errorf("error decoding venue views: %v", err)
	}

	return views, nil
}

// GetHostViewStats returns aggregated view statistics for all venues owned by a host
func (mdb *MongodbRepo) GetHostViewStats(ctx context.Context, hostId string, days int) (*HostViewStats, error) {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}

	stats := &HostViewStats{
		HostID: hostId,
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfWeek := startOfDay.AddDate(0, 0, -int(now.Weekday()))

	// Get total views for all host's venues
	totalCount, err := col.CountDocuments(ctx, bson.M{"host_id": hostId})
	if err != nil {
		return nil, fmt.Errorf("error counting total views: %v", err)
	}
	stats.TotalViews = totalCount

	// Get unique views (by session_id) for host's venues
	uniquePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"host_id": hostId}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$session_id",
		}}},
		{{Key: "$count", Value: "unique_sessions"}},
	}
	uniqueCursor, err := col.Aggregate(ctx, uniquePipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregating unique views: %v", err)
	}
	defer uniqueCursor.Close(ctx)

	var uniqueResult []bson.M
	if err := uniqueCursor.All(ctx, &uniqueResult); err != nil {
		return nil, fmt.Errorf("error decoding unique views: %v", err)
	}
	if len(uniqueResult) > 0 {
		if count, ok := uniqueResult[0]["unique_sessions"].(int32); ok {
			stats.UniqueViews = int64(count)
		}
	}

	// Get views today for host's venues
	todayCount, err := col.CountDocuments(ctx, bson.M{
		"host_id":   hostId,
		"viewed_at": bson.M{"$gte": startOfDay},
	})
	if err != nil {
		return nil, fmt.Errorf("error counting today's views: %v", err)
	}
	stats.ViewsToday = todayCount

	// Get views this week for host's venues
	weekCount, err := col.CountDocuments(ctx, bson.M{
		"host_id":   hostId,
		"viewed_at": bson.M{"$gte": startOfWeek},
	})
	if err != nil {
		return nil, fmt.Errorf("error counting this week's views: %v", err)
	}
	stats.ViewsThisWeek = weekCount

	// Get total number of unique venues for this host
	venuesPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"host_id": hostId}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$venue_id",
		}}},
		{{Key: "$count", Value: "total_venues"}},
	}
	venuesCursor, err := col.Aggregate(ctx, venuesPipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregating venues count: %v", err)
	}
	defer venuesCursor.Close(ctx)

	var venuesResult []bson.M
	if err := venuesCursor.All(ctx, &venuesResult); err != nil {
		return nil, fmt.Errorf("error decoding venues count: %v", err)
	}
	if len(venuesResult) > 0 {
		if count, ok := venuesResult[0]["total_venues"].(int32); ok {
			stats.TotalVenues = int64(count)
		}
	}

	return stats, nil
}

// GetHostViewHistory returns recent view records for all venues owned by a host
func (mdb *MongodbRepo) GetHostViewHistory(ctx context.Context, hostId string, limit int) ([]*VenueView, error) {
	col, err := mdb.GetCollection(ctx, VenueViewsDbName, VenueViewsColName)
	if err != nil {
		return nil, fmt.Errorf("error getting collection: %v", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "viewed_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := col.Find(ctx, bson.M{"host_id": hostId}, opts)
	if err != nil {
		return nil, fmt.Errorf("error finding host views: %v", err)
	}
	defer cursor.Close(ctx)

	var views []*VenueView
	if err := cursor.All(ctx, &views); err != nil {
		return nil, fmt.Errorf("error decoding host views: %v", err)
	}

	return views, nil
}
