// cmd/middleware/auth.go
package middleware

import (
	"context"
	"log"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/gin-gonic/gin"
)

var verifier *oidc.IDTokenVerifier

func InitAuth(issuerURL string) error {
	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return err
	}
	verifier = provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	log.Printf("OIDC verifier initialized (SkipClientIDCheck: true)")
	return nil
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing auth"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == auth {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid format"})
			return
		}

		idToken, err := verifier.Verify(c.Request.Context(), tokenStr)
		if err != nil {
			log.Printf("[AUTH] VERIFY FAILED: %v", err)
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token", "details": err.Error()})
			return
		}

		var claims struct {
			Sub string `json:"sub"`
			Azp string `json:"azp"`
		}
		if err := idToken.Claims(&claims); err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "claim parse failed"})
			return
		}

		// Manually check azp == "frontend"
		if claims.Azp != "frontend" {
			log.Printf("[AUTH] REJECTED: azp=%s (expected 'frontend')", claims.Azp)
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid client"})
			return
		}

		c.Set("user_id", claims.Sub)
		c.Next()
	}
}
