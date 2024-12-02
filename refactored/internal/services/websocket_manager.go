package services

import (
	"errors"
	"log"
	"net/http"

	"planning/internal/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "https://planning.germainleignel.com"
	},
}

func HandleWebSocketConnection(w http.ResponseWriter, r *http.Request, roomID, userName string, isGameMaster bool) error {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Validate roomID
	roomsMu.RLock()
	room, exists := rooms[roomID]
	roomsMu.RUnlock()

	if !exists {
		return errors.New("room not found")
	}

	// Create a new user and add to the room
	user := &models.User{
		Name: userName,
		Conn: conn,
	}

	room.Mu.Lock()
	if isGameMaster {
		if room.GameMaster != "" {
			room.Mu.Unlock()
			return errors.New("room already has a game master")
		}
		room.GameMaster = userName
	}
	room.Users[userName] = user
	room.Mu.Unlock()

	// Broadcast a user join event
	broadcastMessage(room, models.Message{
		Type:    "user_joined",
		Payload: map[string]string{"name": userName},
	})

	// Listen for incoming messages
	for {
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		// Handle the message
		if err := handleWebSocketMessage(room, user, msg); err != nil {
			log.Printf("error handling message: %v", err)
			break
		}
	}

	// Cleanup on disconnect
	room.Mu.Lock()
	delete(room.Users, userName)
	if room.GameMaster == userName {
		room.GameMaster = ""
	}
	room.Mu.Unlock()

	// Broadcast a user left event
	broadcastMessage(room, models.Message{
		Type:    "user_left",
		Payload: map[string]string{"name": userName},
	})

	return nil
}

// broadcastMessage sends a message to all users in the room
func broadcastMessage(room *models.Room, msg models.Message) {
	room.Mu.RLock()
	defer room.Mu.RUnlock()

	for _, user := range room.Users {
		if err := user.Conn.WriteJSON(msg); err != nil {
			log.Printf("error broadcasting message to %s: %v", user.Name, err)
		}
	}
}

// handleWebSocketMessage processes an incoming WebSocket message
func handleWebSocketMessage(room *models.Room, user *models.User, msg models.Message) error {
	switch msg.Type {
	case "vote":
		return handleVote(room, user, msg)
	case "reveal_votes":
		return handleRevealVotes(room, user)
	default:
		return errors.New("unsupported message type")
	}
}

// handleVote processes a vote message
func handleVote(room *models.Room, user *models.User, msg models.Message) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid vote payload")
	}

	vote, ok := payload["vote"].(float64) // JSON numbers are parsed as float64
	if !ok {
		return errors.New("vote not provided or invalid")
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	currentTicket := room.Tickets[room.CurrentTicket]
	currentTicket.Votes[user.Name] = int(vote)

	// Broadcast the vote to the room
	broadcastMessage(room, models.Message{
		Type:    "vote_cast",
		Payload: map[string]interface{}{"name": user.Name, "vote": int(vote)},
	})

	return nil
}

// handleRevealVotes processes a reveal_votes message
func handleRevealVotes(room *models.Room, user *models.User) error {
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if user.Name != room.GameMaster {
		return errors.New("only the game master can reveal votes")
	}

	room.VotesRevealed = true

	// Broadcast the votes to the room
	broadcastMessage(room, models.Message{
		Type:    "votes_revealed",
		Payload: room.Tickets[room.CurrentTicket].Votes,
	})

	return nil
}
