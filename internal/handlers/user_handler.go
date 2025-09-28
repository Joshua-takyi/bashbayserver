package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/models"
	"github.com/joshua-takyi/ww/internal/services"
	"github.com/supabase-community/gotrue-go/types"
)

func CreateUser(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		createdUser, err := u.CreateUser(&user)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, createdUser)
	}
}

func AuthenticateUser(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error(), "message": "invalid request payload"})
			return
		}

		authResponse, err := u.AuthenticateUser(req.Email, req.Password)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error(), "message": "invalid email or password"})
			return
		}

		isProduction := os.Getenv("GIN_MODE") == "production"

		if tokenRes, ok := authResponse.(*types.TokenResponse); ok && tokenRes.AccessToken != "" {
			// Access token
			c.SetCookie(
				"access_token",
				tokenRes.AccessToken,
				tokenRes.ExpiresIn*24*7,
				"/",
				"", // let Gin pick current domain
				isProduction,
				true,
			)

			// Refresh token
			c.SetCookie(
				"refresh_token",
				tokenRes.RefreshToken,
				3600*24*30,
				"/",
				"",
				isProduction,
				true,
			)

			// Return user info but not tokens
			c.JSON(200, gin.H{
				"user": tokenRes.User,
			})
			return
		}

		c.JSON(500, gin.H{"error": "invalid token response"})
	}
}

func GetUser(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get user ID from URL parameter
		id := c.Param("id")
		if id == "" {
			c.JSON(400, gin.H{"error": "user ID is required"})
			return
		}

		// Parse the UUID from the URL
		userId, err := uuid.Parse(id)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid user ID format"})
			return
		}

		// Get claims from context (set by AuthMiddleware)
		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(500, gin.H{"error": "invalid user claims"})
			return
		}

		// Parse the user ID from the claims
		var claimsUserID uuid.UUID
		if claims.UserID != "" {
			claimsUserID, err = uuid.Parse(claims.UserID)
			if err != nil {
				c.JSON(401, gin.H{"error": "invalid user ID in token"})
				return
			}
		}

		// Authorization check: user can access their own data or admin can access any
		if claimsUserID != userId && !claims.IsAdmin() {
			c.JSON(403, gin.H{"error": "access denied"})
			return
		}

		// Get the access token from the cookie
		accessToken, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "access token not found"})
			return
		}

		// Proceed to fetch the user
		user, err := u.GetUser(userId, accessToken)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, user)
	}
}
