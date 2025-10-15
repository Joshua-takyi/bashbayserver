package container

import (
	"log/slog"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/joshua-takyi/ww/internal/models"
	"github.com/joshua-takyi/ww/internal/services"
	"github.com/supabase-community/supabase-go"
	"go.mongodb.org/mongo-driver/mongo"
)

// Container holds all application dependencies
type Container struct {
	Logger     *slog.Logger
	Cloudinary *cloudinary.Cloudinary
	// Database clients
	SupabaseClient    *supabase.Client
	MongoDBClient     *mongo.Client
	UserService       *services.UserService
	VenueService      *services.VenuesService
	FavouritesService *services.FavouriteService
}

// NewContainer creates a new dependency injection container
func NewContainer(
	logger *slog.Logger,
	cloudinary *cloudinary.Cloudinary,
	supabaseClient *supabase.Client,
	mongoDBClient *mongo.Client,
	supaUrl, supaKey string,
) *Container {
	// Initialize repositories
	supa := models.SupabaseNewRepo(supabaseClient, supaUrl, supaKey)
	mongo := models.MongodbNewRepo(mongoDBClient)
	userService := services.NewUserService(supa)
	venueService := services.NewVenuesService(supa, mongo)
	favouriteService := services.NewFavouriteService(mongo)

	return &Container{
		Logger:            logger,
		Cloudinary:        cloudinary,
		SupabaseClient:    supabaseClient,
		MongoDBClient:     mongoDBClient,
		UserService:       userService,
		FavouritesService: favouriteService,
		VenueService:      venueService,
	}
}
