package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joshua-takyi/ww/internal/container"
	"github.com/joshua-takyi/ww/internal/handlers"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/middleware"
)

func SetupRoutes(container *container.Container) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// what this does is to redirect requests with a trailing slash to the same path without the trailing slash.
	// For example, a request to /api/v1/users/ would be redirected to /api/v1/users
	// This is useful for ensuring consistent URL patterns and avoiding duplicate content issues.
	// r.RedirectTrailingSlash = true
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Add middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(container.Logger))
	r.Use(middleware.ErrorHandler(container.Logger))
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "OK",
				"service": "bashbay-api",
			})
		})

		// public routes
		v1.POST("/signup", handlers.CreateUser(container.UserService))
		v1.POST("/login", handlers.AuthenticateUser(container.UserService))
		v1.POST("/logout", handlers.Logout())

		// OAuth routes
		v1.GET("/auth/google", handlers.GoogleAuth(container.UserService))
		v1.GET("/auth/google/callback", handlers.GoogleAuthCallback(container.UserService))

		// Add this to the public routes section (around line 49)
		// v1.GET("/users/:id/public", handlers.GetPublicUserProfile(container.UserService))

		// venues public route
		v1.GET("/venues/search", handlers.QueryVenues(container.VenueService))
		v1.GET("/venues", handlers.ListVenues(container.VenueService))
		v1.GET("/venues/slug/:slug", handlers.GetVenueBySlug(container.VenueService))
		v1.POST("/venues/:id/view", handlers.TrackVenueView(container.VenueService)) // ADD THIS

	}

	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware(container.SupabaseClient, container.UserService, container.Logger))

	userRoutes := protected.Group("/users")
	{
		protected.GET("/profile", func(c *gin.Context) {
			user, exist := c.Get("user")
			if !exist {
				c.JSON(401, gin.H{"error": "Unauthorized"})
				return
			}

			// Cast to EnhancedClaims to access role and other profile data
			enhancedClaims, ok := user.(*helpers.EnhancedClaims)
			if !ok {
				c.JSON(500, gin.H{"error": "Invalid user claims format"})
				return
			}

			c.JSON(200, gin.H{
				"status":       "OK",
				"user_id":      enhancedClaims.UserID,
				"email":        enhancedClaims.Email,
				"role":         enhancedClaims.Role,
				"username":     enhancedClaims.Username,
				"is_admin":     enhancedClaims.IsAdmin(),
				"auth_role":    enhancedClaims.Role, // This is the "authenticated" role from Supabase auth
				"avatar_url":   enhancedClaims.AvatarURL,
				"created_at":   enhancedClaims.CreatedAt,
				"fullname":     enhancedClaims.Fullname,
				"phone_number": enhancedClaims.PhoneNumber,
			})
		})

		userRoutes.GET("/:id", handlers.GetUser(container.UserService))
		userRoutes.PATCH("/:id", handlers.UpdateUser(container.UserService))
		userRoutes.DELETE("/:id", handlers.DeleteUser(container.UserService))
		userRoutes.PATCH("/avatar/:id", handlers.UploadAvatar(container.UserService, container.Cloudinary))
	}

	// Future routes for other services
	// eventRoutes := v1.Group("/events")
	// {
	//     eventRoutes.POST("/", handlers.CreateEvent(container.EventService))
	//     eventRoutes.GET("/", handlers.ListEvents(container.EventService))
	// }
	//
	venueRoutes := protected.Group("/venues")
	{
		venueRoutes.POST("/", handlers.CreateVenueHandler(container.VenueService))
		venueRoutes.GET("/", handlers.ListVenues(container.VenueService))
		venueRoutes.GET("/:id", handlers.ListVenueByID(container.VenueService))
		venueRoutes.DELETE("/:id", handlers.DeleteVenue(container.VenueService))
		venueRoutes.GET("/host-venues/:host_id", handlers.ListVenuesByHost(container.VenueService))
		venueRoutes.POST("/many", handlers.CreateManyVenues(container.VenueService))
		venueRoutes.GET("/:id/stats", handlers.GetVenueViewStats(container.VenueService))
		venueRoutes.GET("/:id/history", handlers.GetVenueViewHistory(container.VenueService))

		// Host analytics routes (efficient queries by host_id)
		venueRoutes.GET("/host/:host_id/analytics", handlers.GetHostViewStats(container.VenueService))
		venueRoutes.GET("/host/:host_id/views", handlers.GetHostViewHistory(container.VenueService))

		// venueRoutes.GET("/search", handlers.QueryVenues(container.VenueService))
		// venueRoutes.PATCH("/:id", handlers.UpdateVenue(container.VenueService))

	}
	//
	// reviewRoutes := v1.Group("/reviews")
	// {
	//     reviewRoutes.POST("/", handlers.CreateReview(container.ReviewService))
	//     reviewRoutes.GET("/", handlers.ListReviews(container.ReviewService))
	// }

	{
		favRoutes := protected.Group("/favourites")
		favRoutes.GET("/", handlers.GetUserFavourites(container.FavouritesService))
		favRoutes.POST("/:id", handlers.AddToFavourites(container.FavouritesService))
		favRoutes.DELETE("/:id", handlers.RemoveFromFavourite(container.FavouritesService))
	}

	return r
}
