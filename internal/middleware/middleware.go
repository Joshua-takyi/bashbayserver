package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/services"
	"github.com/supabase-community/gotrue-go/types"
	"github.com/supabase-community/supabase-go"
)

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// StructuredLogger provides structured logging middleware
func StructuredLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request completion
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		requestID, _ := c.Get("request_id")

		logger.Info("HTTP Request",
			"request_id", requestID,
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency,
			"client_ip", clientIP,
		)
	}
}

// ErrorHandler provides centralized error handling
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID, _ := c.Get("request_id")

			logger.Error("Request error",
				"request_id", requestID,
				"error", err.Error(),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
			)

			// Don't return error details in production
			c.JSON(500, gin.H{
				"error":      "Internal server error",
				"request_id": requestID,
			})
		}
	}
}

// CORS middleware for cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func AuthMiddleware(supabaseClient *supabase.Client, userService *services.UserService, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get JWT token from cookie
		token, err := c.Cookie("access_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized access",
				"error":   "JWT token not found in cookie",
			})
			c.Abort()
			return
		}

		// Validate token using Supabase JWKS
		claims, err := helpers.ValidateToken(token)
		if err != nil {
			// Token validation failed, try to refresh
			refreshToken, refreshErr := c.Cookie("refresh_token")
			if refreshErr != nil {
				// No refresh token, return unauthorized
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "Unauthorized access",
					"error":   err.Error(),
				})
				c.Abort()
				return
			}

			// Try to refresh the token
			refreshResponse, refreshErr := userService.RefreshToken(refreshToken)
			if refreshErr != nil {
				logger.Error("Token refresh failed", "error", refreshErr)
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "Unauthorized access",
					"error":   "Token expired and refresh failed",
				})
				c.Abort()
				return
			}

			// Refresh succeeded, set new cookies
			isProduction := os.Getenv("GIN_MODE") == "production"
			if tokenRes, ok := refreshResponse.(*types.TokenResponse); ok && tokenRes.AccessToken != "" {
				logger.Info("Token refreshed successfully",
					"user_id", tokenRes.User.ID,
					"expires_in", tokenRes.ExpiresIn,
				)
				// Set new access token cookie
				c.SetCookie(
					"access_token",
					tokenRes.AccessToken,
					tokenRes.ExpiresIn,
					"/",
					"", // let Gin pick current domain
					isProduction,
					true,
				)
				// Set new refresh token cookie
				c.SetCookie(
					"refresh_token",
					tokenRes.RefreshToken,
					3600*24*30, // 30 days
					"/",
					"",
					isProduction,
					true,
				)
				// Update token variable with the new access token
				token = tokenRes.AccessToken
				// Validate the new token
				claims, err = helpers.ValidateToken(token)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{
						"message": "Unauthorized access",
						"error":   "Refreshed token validation failed",
					})
					c.Abort()
					return
				}
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "Unauthorized access",
					"error":   "Invalid refresh response",
				})
				c.Abort()
				return
			}
		}

		// Fetch profile data from Supabase using the user service (which uses authenticated client)
		var profileRole, username, fullname, phoneNumber, avatarURL string
		var createdAt time.Time
		userID, parseErr := uuid.Parse(claims.Subject)
		if parseErr != nil {
			logger.Error("Invalid user ID in token", "user_id", claims.Subject, "error", parseErr)
			profileRole = "guest"
			username = ""
		} else {
			user, err := userService.GetUser(userID, token)
			if err != nil {
				logger.Info("Profile not found, using default role",
					"user_id", claims.Subject,
					"error", err,
				)
				profileRole = "guest"
				username = ""
			} else {
				profileRole = user.Role
				if profileRole == "" {
					profileRole = "guest"
					logger.Info("Empty role in profile, defaulting to guest",
						"user_id", claims.Subject,
					)
				}
				phoneNumber = user.PhoneNumber
				fullname = user.FullName
				username = user.Username
				avatarURL = user.AvatarURL
				createdAt = user.CreatedAt
			}
		}

		// Create enhanced claims with profile data
		enhancedClaims := &helpers.EnhancedClaims{
			CustomClaims: claims,
			Role:         profileRole,
			UserID:       claims.Subject,
			Username:     username,
			Email:        claims.Email,
			Fullname:     fullname,
			PhoneNumber:  phoneNumber,
			AvatarURL:    avatarURL,
			CreatedAt:    createdAt.Format(time.RFC3339),
		}

		// Store enhanced claims in context
		c.Set("user", enhancedClaims)
		c.Next()
	}
}

// switch c.Request.Method {
// case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
// 	csrfCookie, err := c.Cookie("csrf_token")
// 	if err != nil {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "CSRF token cookie not found"})
// 		c.Abort()
// 		return
// 	}
// 	csrfHeader := c.GetHeader("X-CSRF-Token")
// 	if csrfHeader == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "CSRF token header not found"})
// 		c.Abort()
// 		return
// 	}

// 	if csrfCookie != csrfHeader {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid CSRF token"})
// 		c.Abort()
// 		return
// 	}
// }

// Continue to the next handler
