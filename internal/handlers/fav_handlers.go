package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/services"
)

func AddToFavourites(f *services.FavouriteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramId := c.Param("id")
		trimedId := helpers.StringTrim(paramId)
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

		userId := userClaims.UserID

		parsedUserId, err := uuid.Parse(userId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		var reqBody struct {
			ItemType string `json:"item_type" binding:"required"`
		}

		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body", "details": err.Error()})
			return
		}

		res, err := f.AddToFavourites(c.Request.Context(), parsedUserId, trimedId, reqBody.ItemType)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func RemoveFromFavourite(f *services.FavouriteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramId := c.Param("id")
		trimedId := helpers.StringTrim(paramId)
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

		userId := userClaims.UserID

		parsedUserId, err := uuid.Parse(userId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		err = f.RemoveFromFavourites(c.Request.Context(), parsedUserId, trimedId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Item removed from favourites"})
	}
}

func GetUserFavourites(f *services.FavouriteService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		userId := userClaims.UserID

		parsedUserId, err := uuid.Parse(userId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		res, err := f.GetFavouritesByUserID(c.Request.Context(), parsedUserId)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}
