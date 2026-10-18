package router

import (
	"net/http"
	"online-judge-lite-backend/internal/config"
	"online-judge-lite-backend/internal/handler"
	"online-judge-lite-backend/internal/middleware"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func New(cfg config.Config, h *handler.Handler) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors(cfg.FrontendOrigins))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "online-judge-lite-api", "time": time.Now().UTC()})
	})
	api := r.Group(cfg.BasePath)
	api.POST("/auth/login", h.Login)
	auth := api.Group("")
	auth.Use(middleware.RequireAuth(cfg.JWTSecret))
	auth.GET("/auth/me", h.Me)
	admin := auth.Group("")
	admin.Use(middleware.RequireRole("ADMIN"))
	admin.GET("/admin/users", h.ListUsers)
	admin.POST("/admin/users", h.CreateUser)
	admin.POST("/admin/users/bulk", h.BulkCreateUsers)
	admin.PUT("/admin/users/:id/reset-password", h.ResetPassword)
	admin.POST("/problem-groups", h.CreateGroup)
	admin.POST("/problems", h.CreateProblem)
	admin.POST("/problems/:id/test-cases", h.AddTestCase)
	auth.GET("/problem-groups", h.ListGroups)
	auth.GET("/problem-groups/:id", h.GetGroup)
	auth.GET("/problems/:id", h.GetProblem)
	auth.POST("/submissions", h.CreateSubmission)
	auth.GET("/submissions/me", h.ListMySubmissions)
	auth.GET("/submissions/:id", h.GetSubmission)
	auth.GET("/scoreboards/groups/:group_id", h.Scoreboard)
	return r
}
func cors(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.TrimRight(o, "/")] = true
	}
	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
