package main

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// Get the Redis address from environment variable
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalf("REDIS_ADDR environment variable is not set")
	}

	// Set up the Redis client
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr, // Using the REDIS_ADDR from environment variable
	})

	// Ping Redis to check if it's reachable
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Printf("Connected to Redis: %v", pong)

	// Attempt to flush data using FlushDB (if FlushAll doesn't work)
	err = client.FlushDB(ctx).Err() // Clears the current database
	if err != nil {
		log.Fatalf("Could not flush Redis: %v", err)
	}

	log.Println("All data has been removed from Redis.")
}
