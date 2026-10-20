package controller

import (
	"online-judge-lite-backend/internal/config"
	"online-judge-lite-backend/internal/runner"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Controller struct {
	DB     *pgxpool.Pool
	Config config.Config
	Queue  chan int
	Runner *runner.Runner
}

func New(db *pgxpool.Pool, cfg config.Config, queue chan int) *Controller {
	return &Controller{DB: db, Config: cfg, Queue: queue , Runner: runner.New(cfg)}
}
func (ctl *Controller) makeToken(id int, username, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{"user_id": id, "username": username, "role": role, "iat": now.Unix(), "exp": now.Add(time.Duration(ctl.Config.JWTTTLHours) * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(ctl.Config.JWTSecret))
}
func uid(c *gin.Context) int      { v, _ := c.Get("user_id"); id, _ := v.(int); return id }
func role(c *gin.Context) string  { v, _ := c.Get("role"); s, _ := v.(string); return s }
func isAdmin(c *gin.Context) bool { return role(c) == "ADMIN" }
