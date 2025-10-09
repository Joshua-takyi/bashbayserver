package handlers

import (
	"net/http"
	"os"
	"strings"

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
			// Access token - expires in 1 hour (3600 seconds)
			c.SetCookie(
				"access_token",
				tokenRes.AccessToken,
				tokenRes.ExpiresIn,
				"/",
				"", // let Gin pick current domain
				isProduction,
				true,
			)

			// Refresh token - expires in 30 days
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

func UpdateUser(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramId := c.Param("id")
		paramId = strings.TrimSpace(paramId)
		paramId = strings.Trim(paramId, " ")
		if paramId == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID is required",
			})
			return
		}

		var user map[string]interface{}
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		claims, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid user claims"})
			return
		}
		userId, err := uuid.Parse(userClaims.UserID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		parsedParamId, err := uuid.Parse(paramId)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		accessToken, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "Access token not found"})
			return
		}

		if userId != parsedParamId && !userClaims.IsAdmin() {
			c.JSON(403, gin.H{"error": "Access denied"})
			return
		}

		data, err := u.UpdateUser(c.Request.Context(), user, parsedParamId, accessToken)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, data)
	}
}

func DeleteUser(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramId := strings.TrimSpace(c.Param("id"))
		if paramId == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID is required",
			})
			return
		}

		claims, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid user claims"})
			return
		}

		parsedParamId, err := uuid.Parse(paramId)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		accessToken, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "Access token not found"})
			return
		}
		if !userClaims.IsAdmin() {
			c.JSON(403, gin.H{"error": "Access denied: only admins can delete users"})
			return
		}

		err = u.DeleteUser(c.Request.Context(), parsedParamId, accessToken)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "user deleted successfully"})
	}
}

func UploadAvatar(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramId := strings.TrimSpace(c.Param("id"))
		if paramId == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "user ID is required",
			})
			return
		}
		var imageData string
		if err := c.ShouldBindJSON(&imageData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		userClaims, ok := claims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid user claims"})
			return
		}
		userId, err := uuid.Parse(userClaims.UserID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		accessToken, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(401, gin.H{"error": "Access token not found"})
			return
		}
		avatarURL, err := u.UploadAvatar(c.Request.Context(), userId, imageData, accessToken)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"avatar_url": avatarURL})
	}
}
