package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"online-judge-lite-backend/internal/config"
	"online-judge-lite-backend/internal/controller"
	"online-judge-lite-backend/internal/database"
	"online-judge-lite-backend/internal/handler"
	"online-judge-lite-backend/internal/router"
	"online-judge-lite-backend/internal/worker"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = runMigration(ctx, db); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	// Jobs marked RUNNING by a previous API process may have been interrupted.
	if _, err = db.Exec(ctx, `UPDATE submissions SET verdict='PENDING' WHERE verdict='RUNNING'`); err != nil {
		log.Fatalf("submission recovery failed: %v", err)
	}
	queue := make(chan int, cfg.QueueSize)
	ctl := controller.New(db, cfg, queue)
	if err = ctl.SeedAdmin(ctx); err != nil {
		log.Fatalf("admin seeding failed: %v", err)
	}
	wp := worker.New(db, cfg, queue)
	wp.Start(ctx)
	h := handler.New(ctl)
	r := router.New(cfg, h)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r, 
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, done := context.WithTimeout(context.Background(), 8*time.Second)
		defer done()
		
		// Sekarang srv adalah *http.Server, jadi method Shutdown() ada!
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v\n", err)
		}
	}()

	// 4. Jalankan server menggunakan srv.ListenAndServe()
	log.Printf("Online Judge API listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
func runMigration(ctx context.Context, db *pgxpool.Pool) error {
	paths := []string{"migrations/001_init.sql", "/app/migrations/001_init.sql"}
	var content []byte
	var err error
	for _, path := range paths {
		content, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, string(content), pgx.QueryExecModeSimpleProtocol)
	return err
}
