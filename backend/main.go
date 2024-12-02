package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var Version = "0.1.1"

type Room struct {
	ID            string
	Tickets       []Ticket
	Users         map[string]*User
	GameMaster    string
	CurrentTicket int
	VotesRevealed bool
	Mu            sync.RWMutex
}

type RoomData struct {
	ID            string
	Tickets       []Ticket
	Users         map[string]string
	GameMaster    string
	CurrentTicket int
	VotesRevealed bool
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
	redisClient *redis.Client
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") == "https://planning.germainleignel.com"
		},
	}
	ErrRoomNotFound   = errors.New("room not found")
	ErrUserExists     = errors.New("user already exists in room")
	ErrInvalidPayload = errors.New("invalid message payload")
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

func logEvent(entry LogEntry) {
	entry.Time = time.Now()
	jsonLog, _ := json.Marshal(entry)
	log.Printf("%s", jsonLog)
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

	roomData := toRoomData(room)
	roomDataBytes, _ := json.Marshal(roomData)
	redisClient.Set(context.Background(), "room:"+roomID, roomDataBytes, 0)

	logEvent(LogEntry{
		Event:  "room_created",
		RoomID: roomID,
	})

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

	roomDataBytes, err := redisClient.Get(context.Background(), "room:"+roomID).Bytes()
	if err == redis.Nil {
		logEvent(LogEntry{Event: "error", Error: "room_not_found", RoomID: roomID})
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var roomData RoomData
	if err := json.Unmarshal(roomDataBytes, &roomData); err != nil {
		log.Printf("Unmarshal error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	room := fromRoomData(roomData)

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

	log.Printf("WebSocket connection initialized for user %s in room %s", userName, roomID)

	conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket closed for user %s in room %s: code=%d, text=%s", userName, roomID, code, text)
		room.Mu.Lock()
		delete(room.Users, userName)
		if userName == room.GameMaster {
			room.GameMaster = ""
			log.Printf("Game master removed from room %s", roomID)
		}
		room.Mu.Unlock()

		roomData = toRoomData(room)
		roomDataBytes, _ = json.Marshal(roomData)
		redisClient.Set(context.Background(), "room:"+roomID, roomDataBytes, 0)

		logEvent(LogEntry{
			Event:  "user_left",
			RoomID: roomID,
			User:   userName,
		})

		broadcastRoomState(room)
		return nil
	})

	user := &User{
		Name: userName,
		Conn: conn,
	}

	room.Mu.Lock()
	if isGameMaster && room.GameMaster == "" {
		room.GameMaster = userName
		log.Printf("Game master set for room %s: %s", roomID, userName)
	}
	room.Users[userName] = user
	room.Mu.Unlock()

	roomData = toRoomData(room)
	roomDataBytes, _ = json.Marshal(roomData)
	redisClient.Set(context.Background(), "room:"+roomID, roomDataBytes, 0)

	logEvent(LogEntry{
		Event:  "user_joined",
		RoomID: roomID,
		User:   userName,
	})

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in WebSocket handler for user %s in room %s: %v\n%s", userName, roomID, r, debug.Stack())
		}

		conn.Close()
		log.Printf("WebSocket connection closed for user %s in room %s", userName, roomID)
	}()

	broadcastRoomState(room)

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s in room %s: %v", userName, roomID, err)
			}
			break
		}

		log.Printf("Received message from user %s in room %s: type=%s", userName, roomID, msg.Type)

		switch msg.Type {
		case "vote":
			handleVote(room, userName, msg)
		case "reveal":
			handleReveal(room, userName)
		case "next":
			handleNext(room, userName)
		default:
			log.Printf("Unknown message type from user %s in room %s: %s", userName, roomID, msg.Type)
		}
	}
}

func handleVote(room *Room, userName string, msg Message) {
	if room.GameMaster == userName {
		log.Printf("Game master attempted to vote: %s", userName)
		return
	}

	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		log.Printf("Invalid vote payload from user %s", userName)
		return
	}

	ticketID, ok1 := payload["ticketId"].(string)
	voteFloat, ok2 := payload["vote"].(float64)
	if !ok1 || !ok2 {
		log.Printf("Missing vote data from user %s: ticketId=%v, vote=%v", userName, ok1, ok2)
		return
	}

	vote := int(voteFloat)

	room.Mu.Lock()
	if ticketID == room.Tickets[room.CurrentTicket].ID {
		for i, ticket := range room.Tickets {
			if ticket.ID == ticketID {
				room.Tickets[i].Votes[userName] = vote
				log.Printf("Vote recorded for user %s on ticket %s: %d", userName, ticketID, vote)
				break
			}
		}
	} else {
		log.Printf("Vote for wrong ticket from user %s: expected=%s, got=%s",
			userName, room.Tickets[room.CurrentTicket].ID, ticketID)
	}
	room.Mu.Unlock()

	roomData := toRoomData(room)
	roomDataBytes, _ := json.Marshal(roomData)
	redisClient.Set(context.Background(), "room:"+room.ID, roomDataBytes, 0)

	broadcastRoomState(room)
}

func handleReveal(room *Room, userName string) {
	if room.GameMaster != userName {
		log.Printf("Non-gamemaster tried to reveal votes: %s", userName)
		return
	}

	room.Mu.Lock()
	room.VotesRevealed = true
	room.Mu.Unlock()

	roomData := toRoomData(room)
	roomDataBytes, _ := json.Marshal(roomData)
	redisClient.Set(context.Background(), "room:"+room.ID, roomDataBytes, 0)

	log.Printf("Votes revealed in room %s by game master %s", room.ID, userName)
	broadcastRoomState(room)
}

func handleNext(room *Room, userName string) {
	if room.GameMaster != userName {
		log.Printf("Non-gamemaster tried to advance ticket: %s", userName)
		return
	}

	room.Mu.Lock()
	oldTicket := room.CurrentTicket
	if room.CurrentTicket < len(room.Tickets)-1 {
		room.CurrentTicket++
		room.VotesRevealed = false
		log.Printf("Advanced to next ticket in room %s: %d -> %d",
			room.ID, oldTicket, room.CurrentTicket)
	} else {
		log.Printf("Attempted to advance past last ticket in room %s", room.ID)
	}
	room.Mu.Unlock()

	roomData := toRoomData(room)
	roomDataBytes, _ := json.Marshal(roomData)
	redisClient.Set(context.Background(), "room:"+room.ID, roomDataBytes, 0)

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
		if user.Conn == nil {
			log.Printf("Nil WebSocket connection for user %s", user.Name)
			continue
		}
		if err := user.Conn.WriteJSON(msg); err != nil {
			log.Printf("Error broadcasting to user %s: %v", user.Name, err)
		}
	}
}

func countUsers() int {
	total := 0
	iter := redisClient.Scan(context.Background(), 0, "room:*", 0).Iterator()
	for iter.Next(context.Background()) {
		roomKey := iter.Val()
		roomDataBytes, err := redisClient.Get(context.Background(), roomKey).Bytes()
		if err != nil {
			log.Printf("Redis error: %v", err)
			continue
		}

		var roomData RoomData
		if err := json.Unmarshal(roomDataBytes, &roomData); err != nil {
			log.Printf("Unmarshal error: %v", err)
			continue
		}

		total += len(roomData.Users)
	}
	if err := iter.Err(); err != nil {
		log.Printf("Redis scan error: %v", err)
	}
	return total
}

func countActiveRooms() int {
	iter := redisClient.Scan(context.Background(), 0, "room:*", 0).Iterator()
	count := 0
	for iter.Next(context.Background()) {
		count++
	}
	if err := iter.Err(); err != nil {
		log.Printf("Redis scan error: %v", err)
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
		users[name] = &User{Name: name}
	}
	return &Room{
		ID:            roomData.ID,
		Tickets:       roomData.Tickets,
		Users:         users,
		GameMaster:    roomData.GameMaster,
		CurrentTicket: roomData.CurrentTicket,
		VotesRevealed: roomData.VotesRevealed,
	}
}

func main() {
	// Initialize Redis client
	initRedisClient()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.LUTC | log.Llongfile)
	log.Printf("Planning Backend v%s starting up", Version)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Create server
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	// Start server
	go func() {
		log.Printf("Server listening on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Fatal server error: %v", err)
		}
	}()

	// Metrics reporting
	go func() {
		for {
			log.Printf("Status - Active rooms: %d, Total users: %d", countActiveRooms(), countUsers())
			time.Sleep(30 * time.Second)
		}
	}()

	// Wait for shutdown signal
	<-stop
	log.Printf("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Printf("Server shutdown complete")
}
