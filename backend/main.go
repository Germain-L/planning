package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID            string
	Tickets       []Ticket
	Users         map[string]*User
	GameMaster    string
	CurrentTicket int
	VotesRevealed bool
	Mu            sync.RWMutex
}

type Ticket struct {
	ID    string
	Votes map[string]int
}

type User struct {
	Name string
	Conn *websocket.Conn
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Error   string      `json:"error,omitempty"`
}

type LogEntry struct {
	Time     time.Time `json:"time"`
	Event    string    `json:"event"`
	RoomID   string    `json:"roomId"`
	User     string    `json:"user,omitempty"`
	TicketID string    `json:"ticketId,omitempty"`
	Vote     int       `json:"vote,omitempty"`
	Error    string    `json:"error,omitempty"`
}

var (
	rooms    = make(map[string]*Room)
	roomsMu  sync.RWMutex
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == "https://planning.germainleignel.com"
		},
	}
	ErrRoomNotFound   = errors.New("room not found")
	ErrUserExists     = errors.New("user already exists in room")
	ErrInvalidPayload = errors.New("invalid message payload")
)

func createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		TicketIDs []string `json:"ticketIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(data.TicketIDs) == 0 {
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

	roomsMu.Lock()
	rooms[roomID] = room
	roomsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"roomId": roomID})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomId")
	userName := r.URL.Query().Get("name")
	isGameMaster := r.URL.Query().Get("gamemaster") == "true"

	if roomID == "" || userName == "" {
		http.Error(w, "Missing roomId or name", http.StatusBadRequest)
		return
	}

	roomsMu.RLock()
	room, exists := rooms[roomID]
	roomsMu.RUnlock()

	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	if _, exists := room.Users[userName]; exists {
		room.Mu.Unlock()
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}
	room.Mu.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	user := &User{
		Name: userName,
		Conn: conn,
	}

	room.Mu.Lock()
	if isGameMaster && room.GameMaster == "" {
		room.GameMaster = userName
	}
	room.Users[userName] = user
	room.Mu.Unlock()

	defer func() {
		conn.Close()
		room.Mu.Lock()
		delete(room.Users, userName)
		if userName == room.GameMaster {
			room.GameMaster = ""
		}
		room.Mu.Unlock()
		broadcastRoomState(room)
	}()

	broadcastRoomState(room)

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		switch msg.Type {
		case "vote":
			handleVote(room, userName, msg)
		case "reveal":
			handleReveal(room, userName)
		case "next":
			handleNext(room, userName)
		}
	}
}

func handleVote(room *Room, userName string, msg Message) {
	if room.GameMaster == userName {
		return
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	ticketID, ok1 := payload["ticketId"].(string)
	voteFloat, ok2 := payload["vote"].(float64)
	if !ok1 || !ok2 {
		return
	}

	vote := int(voteFloat)

	room.Mu.Lock()
	if ticketID == room.Tickets[room.CurrentTicket].ID {
		for i, ticket := range room.Tickets {
			if ticket.ID == ticketID {
				room.Tickets[i].Votes[userName] = vote
				break
			}
		}
	}
	room.Mu.Unlock()

	broadcastRoomState(room)
}

func handleReveal(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	room.Mu.Lock()
	room.VotesRevealed = true
	room.Mu.Unlock()

	broadcastRoomState(room)
}

func handleNext(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	room.Mu.Lock()
	if room.CurrentTicket < len(room.Tickets)-1 {
		room.CurrentTicket++
		room.VotesRevealed = false
	}
	room.Mu.Unlock()

	broadcastRoomState(room)
}

func broadcastRoomState(room *Room) {
	room.Mu.RLock()
	defer room.Mu.RUnlock()

	msg := Message{
		Type:    "roomState",
		Payload: room,
	}

	for _, user := range room.Users {
		user.Conn.WriteJSON(msg)
	}
}

func main() {
	http.HandleFunc("/api/create-room", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://planning.germainleignel.com")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		createRoom(w, r)
	})
	http.HandleFunc("/api/ws", handleWebSocket)

	log.Printf("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
