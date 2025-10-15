package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joshua-takyi/ww/internal/services"
)

// GoogleAuth initiates Google OAuth flow via Supabase
func GoogleAuth(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the redirect URL from query param
		redirectTo := c.Query("redirect_to")
		if redirectTo == "" {
			// Default to frontend URL
			frontendURL := os.Getenv("FRONTEND_URL")
			if frontendURL == "" {
				if os.Getenv("GIN_MODE") == "production" {
					frontendURL = "https://yourdomain.com"
				} else {
					frontendURL = "http://localhost:3000"
				}
			}
			redirectTo = frontendURL + "/auth/callback"
		}

		// Generate the Google OAuth URL via Supabase
		authURL, err := u.GetGoogleAuthURL(redirectTo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate Google auth URL",
				"message": err.Error(),
			})
			return
		}

		// Redirect to Supabase Google OAuth
		c.Redirect(http.StatusTemporaryRedirect, authURL)
	}
}

// GoogleAuthCallback handles the callback from Google OAuth
// Note: With Supabase, tokens are typically sent as URL fragments (#access_token=...)
// which are handled client-side. This endpoint is mainly for error handling.
func GoogleAuthCallback(u *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for errors
		error := c.Query("error")
		errorDescription := c.Query("error_description")

		frontendURL := os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			if os.Getenv("GIN_MODE") == "production" {
				frontendURL = "https://yourdomain.com"
			} else {
				frontendURL = "http://localhost:3000"
			}
		}

		if error != "" {
			// Redirect to frontend with error
			redirectURL := fmt.Sprintf("%s/auth/signin?error=%s&error_description=%s",
				frontendURL, error, errorDescription)
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
			return
		}

		// Supabase sends tokens as URL fragments, which we can't access server-side
		// Redirect to frontend callback page which will handle the tokens
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback")
	}
}

// Logout handler
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		isProduction := os.Getenv("GIN_MODE") == "production"

		// Clear all auth cookies
		c.SetCookie("access_token", "", -1, "/", "", isProduction, true)
		c.SetCookie("refresh_token", "", -1, "/", "", isProduction, true)
		c.SetCookie("session_id", "", -1, "/", "", false, true)

		c.JSON(http.StatusOK, gin.H{
			"message": "Logged out successfully",
		})
	}
}
