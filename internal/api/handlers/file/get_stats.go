package handlers

import (
	"net/http"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services/query"
	"github.com/gin-gonic/gin"
)

func GetMyFileStats(c *gin.Context) {
	userID, exists := userIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	stats, err := query.GetUserFileStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch file stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
