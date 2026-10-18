package middleware

import (
	"net/http"
	"strings"
	"time"

	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token autentikasi diperlukan", nil)
			c.Abort()
			return
		}
		token, err := jwt.ParseWithClaims(parts[1], &Claims{}, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second))
		if err != nil || !token.Valid {
			response.Fail(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token tidak valid atau telah kedaluwarsa", nil)
			c.Abort()
			return
		}
		claims, ok := token.Claims.(*Claims)
		if !ok {
			response.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token tidak valid", nil)
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if !allowed[role.(string)] {
			response.Fail(c, http.StatusForbidden, "FORBIDDEN_ACCESS", "Anda tidak memiliki izin untuk mengakses resource ini", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
func CurrentUser(c *gin.Context) model.Claims {
	id, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	return model.Claims{UserID: id.(int), Username: username.(string), Role: role.(string)}
}
