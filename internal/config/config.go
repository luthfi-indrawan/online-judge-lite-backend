package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env, Port, BasePath, DatabaseURL, JWTSecret string
	FrontendOrigins                             []string
	JWTTTLHours, WorkerCount, QueueSize         int
	AdminUsername, AdminName, AdminPassword     string
	DockerBin, SandboxImage, SandboxWorkdir     string
	CompileTimeoutSeconds                       int
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Env: get("APP_ENV", "development"), Port: get("APP_PORT", "8080"), BasePath: get("APP_BASE_PATH", "/api/v1"),
		DatabaseURL:     get("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/online_judge?sslmode=disable"),
		JWTSecret:       get("JWT_SECRET", "development-only-change-this-secret-before-deploying"),
		FrontendOrigins: split(get("FRONTEND_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		JWTTTLHours:     getInt("JWT_TTL_HOURS", 24), WorkerCount: getInt("WORKER_COUNT", 2), QueueSize: getInt("SUBMISSION_QUEUE_SIZE", 100),
		AdminUsername: get("ADMIN_USERNAME", "admin"), AdminName: get("ADMIN_NAME", "System Administrator"), AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		DockerBin: get("DOCKER_BIN", "docker"), SandboxImage: get("SANDBOX_IMAGE", "online-judge-sandbox:latest"),
		SandboxWorkdir: get("SANDBOX_WORKDIR", "./tmp/submissions"), CompileTimeoutSeconds: getInt("COMPILE_TIMEOUT_SECONDS", 10),
	}
}
func get(k, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fallback
}
func getInt(k string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(k))
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
func split(s string) []string {
	out := []string{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
