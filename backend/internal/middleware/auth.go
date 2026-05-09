package middleware

import (
	"net/http"
	"strings"

	"skoll2/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func JWT(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if authz == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing authorization"})
			return
		}

		token := strings.TrimPrefix(authz, "Bearer ")
		token = strings.TrimSpace(token)
		claims, err := authSvc.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			return
		}

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, ok := c.Get("role")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "no role"})
			return
		}
		if role, ok := roleVal.(string); !ok || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "admin only"})
			return
		}
		c.Next()
	}
}
