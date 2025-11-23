package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/server"
	"honnef.co/go/tools/config"
)

func main() {
	// Load configuration (env vars, defaults…)
	cfg := config.Load()

	// Initialize the HTTP server (Gin, middleware, routes…)
	srv := server.New(cfg)

	// Run server in background goroutine
	go func() {
		log.Printf("🚀 Server starting on %s", cfg.Address())
		if err := srv.Start(); err != nil {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("❌ Shutdown failed: %v", err)
	}

	log.Println("✅ Server stopped cleanly")
}
