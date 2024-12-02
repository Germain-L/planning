package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"planning/internal/handlers"
	"planning/internal/utils"
)

var version = "0.2.0"

func main() {
	// Setup logging
	utils.InitLogger()

	// Routes
	http.HandleFunc("/health", handlers.HealthCheck)
	http.HandleFunc("/api/create-room", handlers.CreateRoom)
	http.HandleFunc("/api/ws", handlers.HandleWebSocket)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := &http.Server{Addr: ":8080"}

	// Start server
	go func() {
		log.Println("Server listening on :8080 in version", version)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server gracefully stopped")
}
