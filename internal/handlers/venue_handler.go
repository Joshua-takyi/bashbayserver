package handlers

import (
	"net/http"
	"strconv"

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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		parsedId, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID in token"})
			return
		}

		createdVenue, err := v.CreateVenue(c.Request.Context(), &venue, parsedId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, createdVenue)
	}
}

func ListVenues(v *services.VenuesService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return
		}
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset parameter"})
			return
		}
		venues, err := v.ListVenues(c.Request.Context(), offsetInt, limitInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, venues)
	}
}
