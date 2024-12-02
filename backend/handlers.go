package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "https://planning.germainleignel.com"
	},
}

func createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logEvent(LogEntry{Event: "error", Error: "method_not_allowed"})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		TicketIDs []string `json:"ticketIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		logEvent(LogEntry{Event: "error", Error: "invalid_request_body"})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(data.TicketIDs) == 0 {
		logEvent(LogEntry{Event: "error", Error: "no_tickets"})
		http.Error(w, "No tickets provided", http.StatusBadRequest)
		return
	}

	roomID := uuid.New().String()
	tickets := make([]Ticket, len(data.TicketIDs))
	for i, id := range data.TicketIDs {
		tickets[i] = Ticket{
			ID:    id,
			Votes: make(map[string]int),
		}
	}

	room := &Room{
		ID:            roomID,
		Tickets:       tickets,
		Users:         make(map[string]*User),
		CurrentTicket: 0,
		VotesRevealed: false,
	}

	saveRoom(room)
	logEvent(LogEntry{Event: "room_created", RoomID: roomID})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"roomId": roomID})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomId")
	userName := r.URL.Query().Get("name")
	isGameMaster := r.URL.Query().Get("gamemaster") == "true"

	if roomID == "" || userName == "" {
		logEvent(LogEntry{Event: "error", Error: "missing_params", RoomID: roomID, User: userName})
		http.Error(w, "Missing roomId or name", http.StatusBadRequest)
		return
	}

	room, err := getRoom(roomID)
	if err != nil {
		logEvent(LogEntry{Event: "error", Error: "room_not_found", RoomID: roomID})
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	if _, exists := room.Users[userName]; exists {
		room.Mu.Unlock()
		logEvent(LogEntry{Event: "error", Error: "user_exists", RoomID: roomID, User: userName})
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}
	room.Mu.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logEvent(LogEntry{Event: "error", Error: "websocket_upgrade_failed", RoomID: roomID, User: userName})
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	setupWebSocketConnection(conn, room, userName, isGameMaster)
}

func deleteAllRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	iter := redisClient.Scan(ctx, 0, "room:*", 0).Iterator()
	delKeys := []string{}

	for iter.Next(ctx) {
		delKeys = append(delKeys, iter.Val())
	}

	if len(delKeys) > 0 {
		redisClient.Del(ctx, delKeys...)
	}

	log.Printf("Deleted %d rooms", len(delKeys))
	w.WriteHeader(http.StatusOK)
}
