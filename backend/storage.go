package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient     *redis.Client
	ErrRoomNotFound = errors.New("room not found")
)

func initRedisClient() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR environment variable not set")
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
}

func getRoom(roomID string) (*Room, error) {
	roomDataBytes, err := redisClient.Get(context.Background(), "room:"+roomID).Bytes()
	if err == redis.Nil {
		return nil, ErrRoomNotFound
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		return nil, err
	}

	var roomData RoomData
	if err := json.Unmarshal(roomDataBytes, &roomData); err != nil {
		return nil, err
	}

	return fromRoomData(roomData), nil
}

func saveRoom(room *Room) error {
	roomData := toRoomData(room)
	roomDataBytes, _ := json.Marshal(roomData)
	return redisClient.Set(context.Background(), "room:"+room.ID, roomDataBytes, 0).Err()
}
