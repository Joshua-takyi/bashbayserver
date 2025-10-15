package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/models"
	"github.com/joshua-takyi/ww/internal/services"
)

func CreateVenueHandler(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var venue models.Venue
		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user claims"})
			return
		}
		if err := c.ShouldBindJSON(&venue); err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse(err.Error()))
			return
		}

		parsedId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		if !claims.IsHost() && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("only users with host role can create venues"))
			return
		}
		accessToken, _ := c.Cookie("access_token")

		createdVenue, err := v.CreateVenue(c.Request.Context(), &venue, parsedId, accessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusCreated, models.SuccessResponse(createdVenue, "Venue created successfully"))
	}
}

func ListVenues(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid limit parameter"))
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid offset parameter"))
			return
		}
		venues, total, err := v.ListVenues(c.Request.Context(), offsetInt, limitInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		page := (offsetInt / limitInt) + 1
		c.JSON(http.StatusOK, models.PaginatedResponse(venues, page, limitInt, total))
	}
}

func ListVenueByID(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		// Normalize incoming id: trim spaces and surrounding quotes which may occur
		// when clients pass values as JSON strings or templates.
		venueID = strings.TrimSpace(venueID)
		venueID = strings.Trim(venueID, "\"'")

		if venueID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue ID is required"))
			return
		}

		// Helpful debug log when parse fails locally
		parsedId, err := uuid.Parse(venueID)
		if err != nil {
			fmt.Printf("failed to parse venue id: %q, error: %v\n", venueID, err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid venue ID format"))
			return
		}

		venue, err := v.ListVenueByID(c.Request.Context(), parsedId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}
		if venue == nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse("venue not found"))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(venue, ""))
	}
}

func DeleteVenue(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		venueID = strings.TrimSpace(venueID)
		venueID = strings.Trim(venueID, "\"'")
		if venueID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue ID is required"))
			return
		}

		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedId, err := uuid.Parse(venueID)
		if err != nil {
			fmt.Printf("failed to parse venue id for delete: %q, error: %v\n", venueID, err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid venue ID format"))
			return
		}

		// Get the venue first to verify ownership
		venue, err := v.ListVenueByID(c.Request.Context(), parsedId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}
		if venue == nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse("venue not found"))
			return
		}

		// Check if the user owns the venue
		if venue.HostId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("forbidden: you can only delete your own venues"))
			return
		}

		// Extract access token cookie to allow repo to perform the delete under the user's session
		accessToken, _ := c.Cookie("access_token")

		if err := v.DeleteVenue(c.Request.Context(), userId, parsedId, accessToken); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "venue deleted successfully"))
	}
}

func ListVenuesByHost(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid limit parameter"))
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid offset parameter"))
			return
		}

		hostID := c.Param("host_id")
		if hostID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("host ID is required"))
			return
		}

		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedId, err := uuid.Parse(hostID)
		if err != nil {
			fmt.Printf("failed to parse host id: %q, error: %v\n", hostID, err)
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid host ID format"))
			return
		}

		// Check if the user is authorized to list venues for this host
		if parsedId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("unauthorized access"))
			return
		}

		accessToken, _ := c.Cookie("access_token")
		vD, total, err := v.ListVenuesByHost(c.Request.Context(), parsedId, offsetInt, limitInt, accessToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("failed to get host venues documents"))
			return
		}

		page := (offsetInt / limitInt) + 1
		c.JSON(http.StatusOK, models.PaginatedResponse(vD, page, limitInt, total))
	}
}

func QueryVenues(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid limit parameter"))
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid offset parameter"))
			return
		}

		// Build query map from request parameters
		query := make(map[string]interface{})

		// Venue type filter
		if venueType := c.Query("venue_type"); venueType != "" {
			query["venue_type"] = venueType
		}

		// Price range filters
		if minPrice := c.Query("min_price"); minPrice != "" {
			if minPriceFloat, err := strconv.ParseFloat(minPrice, 64); err == nil {
				query["min_price"] = minPriceFloat
			}
		}
		if maxPrice := c.Query("max_price"); maxPrice != "" {
			if maxPriceFloat, err := strconv.ParseFloat(maxPrice, 64); err == nil {
				query["max_price"] = maxPriceFloat
			}
		}

		// Capacity range filters
		if minCapacity := c.Query("min_capacity"); minCapacity != "" {
			if minCapInt, err := strconv.Atoi(minCapacity); err == nil {
				query["min_capacity"] = minCapInt
			}
		}
		if maxCapacity := c.Query("max_capacity"); maxCapacity != "" {
			if maxCapInt, err := strconv.Atoi(maxCapacity); err == nil {
				query["max_capacity"] = maxCapInt
			}
		}

		// Location filter (partial match)
		if location := c.Query("location"); location != "" {
			query["location"] = location
		}

		// Amenities filter (comma-separated)
		if amenities := c.Query("amenities"); amenities != "" {
			query["amenities"] = strings.Split(amenities, ",")
		}

		// Status filter
		if status := c.Query("status"); status != "" {
			query["status"] = status
		}

		// Name filter (partial match)
		if name := c.Query("name"); name != "" {
			query["name"] = name
		}

		// Description filter (partial match)
		if description := c.Query("description"); description != "" {
			query["description"] = description
		}

		// If no query parameters provided, return bad request
		if len(query) == 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("at least one query parameter is required"))
			return
		}

		venues, total, err := v.QueryVenues(c.Request.Context(), query, offsetInt, limitInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		page := (offsetInt / limitInt) + 1
		c.JSON(http.StatusOK, models.PaginatedResponse(venues, page, limitInt, total))
	}
}

func GetVenueBySlug(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		slug = helpers.StringTrim(slug)

		if slug == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue slug is required"))
			return
		}

		venue, err := v.GetVenueBySlug(c.Request.Context(), slug)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}
		if venue == nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse("venue not found"))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(venue, ""))
	}
}

func CreateManyVenues(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var venues []*models.Venue
		if err := c.ShouldBindJSON(&venues); err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid request body"))
			return
		}
		if len(venues) == 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("no venues to create"))
			return
		}

		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		if !claims.IsHost() && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("only users with host role can create venues"))
			return
		}

		accessToken, _ := c.Cookie("access_token")

		createdVenues, err := v.CreateManyVenues(c.Request.Context(), venues, userId, accessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusCreated, models.SuccessResponse(createdVenues, "Venues created successfully"))
	}
}

// TrackVenueView records when someone views a venue (with rate limiting)
func TrackVenueView(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		venueID = strings.TrimSpace(venueID)

		if venueID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue ID is required"))
			return
		}

		parsedId, err := uuid.Parse(venueID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid venue ID format"))
			return
		}

		// Get user ID if authenticated (optional)
		var userId *uuid.UUID
		if userClaims, exists := c.Get("user"); exists {
			if claims, ok := userClaims.(*helpers.EnhancedClaims); ok {
				if uid, err := uuid.Parse(claims.UserID); err == nil {
					userId = &uid
				}
			}
		}

		// Get or create session ID
		sessionId, err := c.Cookie("session_id")
		if err != nil || sessionId == "" {
			sessionId = uuid.New().String()
			c.SetCookie("session_id", sessionId, 86400*30, "/", "", false, true) // 30 days
		}

		// Basic bot detection
		userAgent := c.Request.UserAgent()
		if userAgent == "" || strings.Contains(strings.ToLower(userAgent), "bot") ||
			strings.Contains(strings.ToLower(userAgent), "crawler") ||
			strings.Contains(strings.ToLower(userAgent), "spider") {
			// Still return success to not reveal bot detection
			c.JSON(http.StatusOK, models.SuccessResponse(nil, "View tracked successfully"))
			return
		}

		// Track the view
		ipAddress := c.ClientIP()

		// Run tracking in background to not block response
		go func() {
			bgCtx := context.Background()
			if err := v.TrackVenueView(bgCtx, parsedId, userId, sessionId, ipAddress, userAgent); err != nil {
				// Log error but don't expose to client
				fmt.Printf("Failed to track view for venue %s: %v\n", venueID, err)
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "View tracked successfully"))
	}
}

// GetVenueViewStats returns analytics for a venue (host only)
func GetVenueViewStats(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		venueID = strings.TrimSpace(venueID)

		if venueID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue ID is required"))
			return
		}

		parsedVenueId, err := uuid.Parse(venueID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid venue ID format"))
			return
		}

		// Get days parameter (default 30)
		days := 30
		if daysParam := c.Query("days"); daysParam != "" {
			if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
				days = d
			}
		}

		stats, err := v.GetVenueViewStats(c.Request.Context(), parsedVenueId, days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(stats, ""))
	}
}

// GetVenueViewHistory returns recent view records (host only)
func GetVenueViewHistory(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		venueID = strings.TrimSpace(venueID)

		if venueID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("venue ID is required"))
			return
		}

		parsedVenueId, err := uuid.Parse(venueID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid venue ID format"))
			return
		}

		// Get limit parameter (default 100)
		limit := 100
		if limitParam := c.Query("limit"); limitParam != "" {
			if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
				limit = l
			}
		}

		history, err := v.GetVenueViewHistory(c.Request.Context(), parsedVenueId, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(history, ""))
	}
}

// GetHostViewStats returns aggregated analytics for all venues owned by a host
func GetHostViewStats(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		hostID := c.Param("host_id")
		hostID = strings.TrimSpace(hostID)

		if hostID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("host ID is required"))
			return
		}

		// Verify user is authorized to view this host's stats
		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedHostId, err := uuid.Parse(hostID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid host ID format"))
			return
		}

		// Check if user is the host or admin
		if parsedHostId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("access denied"))
			return
		}

		// Get days parameter (default 30)
		days := 30
		if daysParam := c.Query("days"); daysParam != "" {
			if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
				days = d
			}
		}

		stats, err := v.GetHostViewStats(c.Request.Context(), parsedHostId, days)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(stats, ""))
	}
}

// GetHostViewHistory returns recent view records for all venues owned by a host
func GetHostViewHistory(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		hostID := c.Param("host_id")
		hostID = strings.TrimSpace(hostID)

		if hostID == "" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("host ID is required"))
			return
		}

		// Verify user is authorized to view this host's history
		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedHostId, err := uuid.Parse(hostID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("invalid host ID format"))
			return
		}

		// Check if user is the host or admin
		if parsedHostId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, models.ErrorResponse("access denied"))
			return
		}

		// Get limit parameter (default 100)
		limit := 100
		if limitParam := c.Query("limit"); limitParam != "" {
			if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
				limit = l
			}
		}

		history, err := v.GetHostViewHistory(c.Request.Context(), parsedHostId, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, models.SuccessResponse(history, ""))
	}
}
