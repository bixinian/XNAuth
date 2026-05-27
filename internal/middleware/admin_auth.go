package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"xnauth/internal/auth"
	"xnauth/pkg/response"
)

const (
	ContextAdminID  = "admin_id"
	ContextUsername = "admin_username"
)

func AdminAuth(manager *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing token")
			c.Abort()
			return
		}

		token := strings.TrimSpace(header[len("Bearer "):])
		claims, err := manager.Parse(token)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid token")
			c.Abort()
			return
		}

		c.Set(ContextAdminID, claims.AdminID)
		c.Set(ContextUsername, claims.Username)
		c.Next()
	}
}
