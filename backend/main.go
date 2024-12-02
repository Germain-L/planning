package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version = "0.1.1"

func main() {
	initRedisClient()
	log.SetFlags(log.LstdFlags | log.LUTC | log.Llongfile)
	log.Printf("Planning Backend v%s starting up", Version)

	setupRoutes()
	server := setupServer()

	// Metrics reporting
	go reportMetrics()

	// Graceful shutdown
	handleShutdown(server)
}

func setupRoutes() {
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/api/create-room", corsMiddleware(createRoom))
	http.HandleFunc("/api/ws", handleWebSocket)
	http.HandleFunc("/api/admin/delete-rooms", adminMiddleware(deleteAllRooms))
}

func setupServer() *http.Server {
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	go func() {
		log.Printf("Server listening on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Fatal server error: %v", err)
		}
	}()

	return server
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Printf("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Printf("Server shutdown complete")
}

func reportMetrics() {
	for {
		log.Printf("Status - Active rooms: %d, Total users: %d",
			countActiveRooms(), countUsers())
		time.Sleep(30 * time.Second)
	}
}
