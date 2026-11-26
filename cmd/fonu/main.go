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

	runDone:=make(chan error,1)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	go func(){runDone<-application.Run()}()
	var runErr error
	select{
	case <-stop:
	case runErr=<-runDone:
		if errors.Is(runErr,http.ErrServerClosed){runErr=nil}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := application.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	if runErr!=nil{log.Fatalf("server error: %v",runErr)}
}
