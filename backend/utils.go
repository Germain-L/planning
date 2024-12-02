package main

import (
	"context"
	"encoding/json"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func logEvent(entry LogEntry) {
	entry.Time = time.Now()
	jsonLog, _ := json.Marshal(entry)
	log.Printf("%s", jsonLog)
}

func handlePanic(conn *websocket.Conn, userName string, roomID string) {
	if r := recover(); r != nil {
		log.Printf("Panic in WebSocket handler for user %s in room %s: %v\n%s",
			userName, roomID, r, debug.Stack())
	}
	conn.Close()
}

func setupCloseHandler(conn *websocket.Conn, room *Room, userName string) {
	conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket closed for user %s in room %s: code=%d, text=%s",
			userName, room.ID, code, text)

		room.Mu.Lock()
		delete(room.Users, userName)
		if userName == room.GameMaster {
			room.GameMaster = ""
		}
		room.Mu.Unlock()

		saveRoom(room)
		logEvent(LogEntry{Event: "user_left", RoomID: room.ID, User: userName})
		broadcastRoomState(room)
		return nil
	})
}

func countUsers() int {
	total := 0
	iter := redisClient.Scan(context.Background(), 0, "room:*", 0).Iterator()
	for iter.Next(context.Background()) {
		roomKey := iter.Val()
		room, err := getRoom(roomKey[5:]) // Remove "room:" prefix
		if err != nil {
			continue
		}
		total += len(room.Users)
	}
	return total
}

func countActiveRooms() int {
	count := 0
	iter := redisClient.Scan(context.Background(), 0, "room:*", 0).Iterator()
	for iter.Next(context.Background()) {
		count++
	}
	return count
}

func toRoomData(room *Room) RoomData {
	users := make(map[string]string)
	for name, user := range room.Users {
		users[name] = user.Name
	}
	return RoomData{
		ID:            room.ID,
		Tickets:       room.Tickets,
		Users:         users,
		GameMaster:    room.GameMaster,
		CurrentTicket: room.CurrentTicket,
		VotesRevealed: room.VotesRevealed,
	}
}

func fromRoomData(roomData RoomData) *Room {
	users := make(map[string]*User)
	for name := range roomData.Users {
		users[name] = nil // Initialize without connection, will be set when user connects
	}
	return &Room{
		ID:            roomData.ID,
		Tickets:       roomData.Tickets,
		Users:         users,
		GameMaster:    roomData.GameMaster,
		CurrentTicket: roomData.CurrentTicket,
		VotesRevealed: roomData.VotesRevealed,
		Mu:            sync.RWMutex{},
	}
}
