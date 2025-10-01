package handlers

import (
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
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse(err.Error()))
			return
		}

		parsedId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid user ID in token"))
			return
		}

		if !claims.IsHost() && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, helpers.ErrorResponse("only users with host role can create venues"))
			return
		}
		accessToken, _ := c.Cookie("access_token")

		createdVenue, err := v.CreateVenue(c.Request.Context(), &venue, parsedId, accessToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusCreated, helpers.SuccessResponse(createdVenue, "Venue created successfully"))
	}
}

func ListVenues(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid limit parameter"))
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid offset parameter"))
			return
		}
		venues, total, err := v.ListVenues(c.Request.Context(), offsetInt, limitInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err.Error()))
			return
		}

		page := (offsetInt / limitInt) + 1
		c.JSON(http.StatusOK, helpers.PaginatedResponse(venues, page, limitInt, total))
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
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("venue ID is required"))
			return
		}

		// Helpful debug log when parse fails locally
		parsedId, err := uuid.Parse(venueID)
		if err != nil {
			fmt.Printf("failed to parse venue id: %q, error: %v\n", venueID, err)
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid venue ID format"))
			return
		}

		venue, err := v.ListVenueByID(c.Request.Context(), parsedId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err.Error()))
			return
		}
		if venue == nil {
			c.JSON(http.StatusNotFound, helpers.ErrorResponse("venue not found"))
			return
		}

		c.JSON(http.StatusOK, helpers.SuccessResponse(venue, ""))
	}
}

func DeleteVenue(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		venueID := c.Param("id")
		venueID = strings.TrimSpace(venueID)
		venueID = strings.Trim(venueID, "\"'")
		if venueID == "" {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("venue ID is required"))
			return
		}

		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, helpers.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedId, err := uuid.Parse(venueID)
		if err != nil {
			fmt.Printf("failed to parse venue id for delete: %q, error: %v\n", venueID, err)
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid venue ID format"))
			return
		}

		// Get the venue first to verify ownership
		venue, err := v.ListVenueByID(c.Request.Context(), parsedId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err.Error()))
			return
		}
		if venue == nil {
			c.JSON(http.StatusNotFound, helpers.ErrorResponse("venue not found"))
			return
		}

		// Check if the user owns the venue
		if venue.HostId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, helpers.ErrorResponse("forbidden: you can only delete your own venues"))
			return
		}

		// Extract access token cookie to allow repo to perform the delete under the user's session
		accessToken, _ := c.Cookie("access_token")

		if err := v.DeleteVenue(c.Request.Context(), userId, parsedId, accessToken); err != nil {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err.Error()))
			return
		}

		c.JSON(http.StatusOK, helpers.SuccessResponse(nil, "venue deleted successfully"))
	}
}

func ListVenuesByHost(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid limit parameter"))
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid offset parameter"))
			return
		}

		hostID := c.Param("host_id")
		if hostID == "" {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("host ID is required"))
			return
		}

		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, helpers.ErrorResponse("unauthorized"))
			return
		}

		claims, ok := userClaims.(*helpers.EnhancedClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, helpers.ErrorResponse("invalid user claims"))
			return
		}

		userId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid user ID in token"))
			return
		}

		parsedId, err := uuid.Parse(hostID)
		if err != nil {
			fmt.Printf("failed to parse host id: %q, error: %v\n", hostID, err)
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("invalid host ID format"))
			return
		}

		// Check if the user is authorized to list venues for this host
		if parsedId != userId && !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, helpers.ErrorResponse("unauthorized access"))
			return
		}

		accessToken, _ := c.Cookie("access_token")
		vD, total, err := v.ListVenuesByHost(c.Request.Context(), parsedId, offsetInt, limitInt, accessToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, helpers.ErrorResponse("failed to get host venues documents"))
			return
		}

		page := (offsetInt / limitInt) + 1
		c.JSON(http.StatusOK, helpers.PaginatedResponse(vD, page, limitInt, total))
	}
}
