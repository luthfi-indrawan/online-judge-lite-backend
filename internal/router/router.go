package router

import (
	"online-judge-lite-backend/internal/config"
	"online-judge-lite-backend/internal/handler"
	"online-judge-lite-backend/internal/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(cfg config.Config, h *handler.Handler) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// CORS HARUS DIURUTAN PALING PERTAMA SEBELUM APAPUN
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "Origin", "X-Requested-With", "ngrok-skip-browser-warning"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gin.Logger(), gin.Recovery())

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

	auth.POST("/run", h.RunCode)

	auth.POST("/submissions", h.CreateSubmission)
	auth.GET("/submissions/me", h.ListMySubmissions)
	auth.GET("/submissions/:id", h.GetSubmission)
	auth.GET("/scoreboards/groups/:group_id", h.Scoreboard)


	return r
}