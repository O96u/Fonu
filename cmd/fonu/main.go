package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fonu/fonu/internal/app"
	"github.com/fonu/fonu/internal/config"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	cfg := config.Load()
	migrationsDir, err := app.ResolveMigrationsDir()
	if err != nil {
		log.Fatalf("resolve migrations: %v", err)
	}

	staticFS, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatalf("load static files: %v", err)
	}

	application, err := app.New(cfg, staticFS, migrationsDir)
	if err != nil {
		log.Fatalf("start application: %v", err)
	}

	go func() {
		if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := application.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
