package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joshua-takyi/ww/internal/container"
	"github.com/joshua-takyi/ww/internal/handlers"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/middleware"
)

// SetupRoutes configures all routes with the dependency container
func SetupRoutes(container *container.Container) *gin.Engine {
	// Set Gin mode for production
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
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
	// r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	// API version 1
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
				"status":    "OK",
				"user_id":   enhancedClaims.UserID,
				"email":     enhancedClaims.Email,
				"role":      enhancedClaims.Role,
				"username":  enhancedClaims.Username,
				"is_admin":  enhancedClaims.IsAdmin(),
				"auth_role": enhancedClaims.Role, // This is the "authenticated" role from Supabase auth
			})
		})

		userRoutes.GET("/:id", handlers.GetUser(container.UserService))
		userRoutes.PATCH("/:id", handlers.UpdateUser(container.UserService))
		userRoutes.DELETE("/:id", handlers.DeleteUser(container.UserService))
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

	}
	//
	// reviewRoutes := v1.Group("/reviews")
	// {
	//     reviewRoutes.POST("/", handlers.CreateReview(container.ReviewService))
	//     reviewRoutes.GET("/", handlers.ListReviews(container.ReviewService))
	// }

	return r
}
