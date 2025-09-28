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
	SupabaseClient *supabase.Client
	MongoDBClient  *mongo.Client

	// Services (start with what you have, expand later)
	UserService *services.UserService
	// EventService  *services.EventService   // Add these as you create them
	VenueService *services.VenuesService // Add these as you create them
	// ReviewService *services.ReviewService  // Add these as you create them
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
	userRepo := models.SupabaseNewRepo(supabaseClient, supaUrl, supaKey)

	// Initialize services with their respective repositories
	userService := services.NewUserService(userRepo)
	venueService := services.NewVenuesService(userRepo) // userRepo also implements VenuesRepo

	return &Container{
		Logger:         logger,
		Cloudinary:     cloudinary,
		SupabaseClient: supabaseClient,
		MongoDBClient:  mongoDBClient,
		UserService:    userService,
		VenueService:   venueService,
	}
}
