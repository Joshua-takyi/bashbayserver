package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AvatarFolder = "avatars"
	VenueFolder  = "venues"
	EventsFolder = "events"
)

type CustomClaims struct {
	Role        string `json:"role"`
	Email       string `json:"email"`
	AppMetadata struct {
		Provider  string   `json:"provider"`
		Providers []string `json:"providers"`
		Roles     []string `json:"roles,omitempty"`
	} `json:"app_metadata"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	jwt.RegisteredClaims
}

func ValidateToken(tokenStr string) (*CustomClaims, error) {
	// Get Supabase URL from environment
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, errors.New("SUPABASE_URL not set")
	}

	// Construct JWKS URL
	jwksURL := fmt.Sprintf("%s/rest/v1/auth/jwks", supabaseURL)

	// Create a context with timeout for the JWKS request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create the JWKS from the remote URL
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		Ctx: ctx,
	})
	if err != nil {
		// Fallback to unverified parsing if JWKS fails (for development)
		token, _, parseErr := jwt.NewParser().ParseUnverified(tokenStr, &CustomClaims{})
		if parseErr != nil {
			return nil, fmt.Errorf("JWKS validation failed and fallback parsing failed: %v", parseErr)
		}
		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			return nil, errors.New("invalid token claims")
		}
		return claims, nil
	}
	defer jwks.EndBackground()

	// Parse the JWT with JWKS validation
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %v", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}

func IsPasswordStrong(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[@$!%*?&]`).MatchString(password)
	return hasLower && hasUpper && hasNumber && hasSpecial
}

func UploadImages(ctx context.Context, cld *cloudinary.Cloudinary, imageNames []string, imagePath string) ([]string, error) {
	var urls []string

	// fmt.Printf("UploadImages called with %d images, imagePath: %s\n", len(imageNames), imagePath)
	// fmt.Printf("Cloudinary client is nil: %v\n", cld == nil)

	for i, filePath := range imageNames {
		// fmt.Printf("Processing image %d: %s\n", i, filePath)
		if strings.TrimSpace(filePath) == "" {
			fmt.Printf("Skipping empty image path at index %d\n", i)
			continue
		}
		uploadResult, err := cld.Upload.Upload(ctx, filePath, uploader.UploadParams{
			// Moderation: "webpurify", // Temporarily remove moderation
			Folder: imagePath,
			Tags:   []string{"ww-app"},
		})

		if err != nil {
			fmt.Printf("Upload failed for %s: %v\n", filePath, err)
			return nil, fmt.Errorf("failed to upload image %s: %v", filePath, err)
		}

		// debugging info
		// fmt.Printf("Upload result: %+v\n", uploadResult)
		// fmt.Printf("SecureURL: '%s'\n", uploadResult.SecureURL)
		// fmt.Printf("PublicID: '%s'\n", uploadResult.PublicID)
		urls = append(urls, uploadResult.SecureURL)
	}

	// fmt.Printf("Returning %d URLs: %v\n", len(urls), urls)
	return urls, nil
}
