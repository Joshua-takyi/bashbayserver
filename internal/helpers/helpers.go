package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/golang-jwt/jwt/v5"
	// "github.com/joshua-takyi/ww/internal/models"
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

var (
	profanityOnce      sync.Once
	profanityRegex     *regexp.Regexp
	reNonWordSpaceDash = regexp.MustCompile(`[\s\W-]+`)
	reLower            = regexp.MustCompile(`[a-z]`)
	reUpper            = regexp.MustCompile(`[A-Z]`)
	reDigit            = regexp.MustCompile(`\d`)
	reSpecial          = regexp.MustCompile(`[@$!%*?&]`)
)

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
	hasLower := reLower.MatchString(password)
	hasUpper := reUpper.MatchString(password)
	hasNumber := reDigit.MatchString(password)
	hasSpecial := reSpecial.MatchString(password)
	return hasLower && hasUpper && hasNumber && hasSpecial
}

func UploadImages(ctx context.Context, cld *cloudinary.Cloudinary, imageNames []string, imagePath string) ([]string, []string, error) {
	var urls []string
	var publicIDs []string

	for i, filePath := range imageNames {
		// fmt.Printf("Processing image %d: %s\n", i, filePath)
		if strings.TrimSpace(filePath) == "" {
			fmt.Printf("Skipping empty image path at index %d\n", i)
			continue
		}
		uploadResult, err := cld.Upload.Upload(ctx, filePath, uploader.UploadParams{
			Folder: imagePath,
			Tags:   []string{"ww-app"},
		})

		if err != nil {
			fmt.Printf("Upload failed for %s: %v\n", filePath, err)
			return nil, nil, fmt.Errorf("failed to upload image %s: %v", filePath, err)
		}

		urls = append(urls, uploadResult.SecureURL)
		publicIDs = append(publicIDs, uploadResult.PublicID)
	}

	// fmt.Printf("Returning %d URLs: %v\n", len(urls), urls)
	return urls, publicIDs, nil
}

func DeleteImages(ctx context.Context, cld *cloudinary.Cloudinary, folderName string, publicIDs []string) error {
	for _, rawID := range publicIDs {
		publicID := strings.TrimSpace(rawID)
		if publicID == "" {
			continue
		}

		// Ensure folder prefix if provided
		if folderName != "" && !strings.HasPrefix(publicID, folderName+"/") {
			publicID = fmt.Sprintf("%s/%s", folderName, publicID)
		}

		// Attempt deletion
		resp, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
			PublicID: publicID,
		})
		if err != nil {
			fmt.Printf("[Cloudinary] Error deleting '%s': %v\n", publicID, err)
			continue
		}

		switch resp.Result {
		case "ok":
			fmt.Printf("[Cloudinary] Deleted: %s\n", publicID)
		case "not found":
			fmt.Printf("[Cloudinary] Not found: %s\n", publicID)
		default:
			fmt.Printf("[Cloudinary] Unexpected result for '%s': %s\n", publicID, resp.Result)
		}
	}

	return nil
}

func GenerateSlug(name, location string) string {
	combined := fmt.Sprintf("%s %s", name, location)
	slug := strings.ToLower(strings.TrimSpace(combined))
	slug = reNonWordSpaceDash.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func StringTrim(s string) string {
	s = strings.TrimSpace(s)
	return s
}

func RemoveDuplicates(features []string) []string {
	featureMap := make(map[string]bool)
	var uniqueFeatures []string
	for _, feature := range features {
		if _, exists := featureMap[feature]; !exists {
			featureMap[feature] = true
			uniqueFeatures = append(uniqueFeatures, feature)
		}
	}
	return uniqueFeatures
}

func RemoveProfanity(input string) string {
	re := getProfanityRegex()
	if re == nil {
		return input
	}
	return re.ReplaceAllStringFunc(input, func(matched string) string {
		return strings.Repeat("*", len(matched))
	})
}

func getProfanityRegex() *regexp.Regexp {
	profanityOnce.Do(func() {
		words := loadProfanityList()
		if len(words) == 0 {
			profanityRegex = nil
			return
		}

		// Quote each word for regex, join with alternation, and add word boundaries.
		// Case-insensitive via (?i). Using \b to reduce false positives inside other words.
		escaped := make([]string, 0, len(words))
		for _, w := range words {
			w = strings.TrimSpace(w)
			if w == "" {
				continue
			}
			escaped = append(escaped, regexp.QuoteMeta(w))
		}
		if len(escaped) == 0 {
			profanityRegex = nil
			return
		}

		pattern := `(?i)\b(` + strings.Join(escaped, `|`) + `)\b`
		// Compile safely; on error, disable filtering rather than panicking.
		if re, err := regexp.Compile(pattern); err == nil {
			profanityRegex = re
		} else {
			fmt.Printf("[profanity] failed to compile regex: %v\n", err)
			profanityRegex = nil
		}
	})
	return profanityRegex
}

// loadProfanityList loads words from PROFANITY_FILE or PROFANITY_WORDS, de-duplicates, and lowercases.
func loadProfanityList() []string {
	// Attempt file first
	var all []string
	filePath := strings.TrimSpace(os.Getenv("PROFANITY_FILE"))
	candidatePaths := []string{}
	if filePath != "" {
		candidatePaths = append(candidatePaths, filePath)
	}
	// Default fallback relative to server root
	candidatePaths = append(candidatePaths, "config/profanity.txt")

	for _, path := range candidatePaths {
		if path == "" {
			continue
		}
		if data, err := os.ReadFile(path); err == nil {
			all = append(all, normalizeWordList(strings.Split(string(data), "\n"), true)...)
			break
		} else {
			// only log if an explicit path was provided
			if path == filePath && filePath != "" {
				fmt.Printf("[profanity] failed to read file '%s': %v\n", path, err)
			}
		}
	}

	// Then env list (comma-separated)
	if csv := strings.TrimSpace(os.Getenv("PROFANITY_WORDS")); csv != "" {
		all = append(all, normalizeWordList(strings.Split(csv, ","), false)...)
	}

	// De-duplicate (case-insensitive)
	if len(all) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(all))
	uniq := make([]string, 0, len(all))
	for _, w := range all {
		key := strings.ToLower(w)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniq = append(uniq, w)
	}
	return uniq
}

// normalizeWordList trims tokens, skips blanks and (if enabled) comment lines.
func normalizeWordList(tokens []string, allowComments bool) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if allowComments && strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, t)
	}
	return out
}
