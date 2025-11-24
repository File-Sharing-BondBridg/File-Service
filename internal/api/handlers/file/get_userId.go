package handlers

import "github.com/gin-gonic/gin"

func userIDFromContext(c *gin.Context) (string, bool) {
	id, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return id.(string), true
}
