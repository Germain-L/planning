package main

import (
	"log"

	"github.com/gorilla/websocket"
)

func setupWebSocketConnection(conn *websocket.Conn, room *Room, userName string, isGameMaster bool) {
	user := &User{Name: userName, Conn: conn}
	activeConnections.Store(room.ID+":"+userName, conn)

	room.Mu.Lock()
	room.Users[userName] = user
	if isGameMaster {
		room.GameMaster = userName
	}
	room.Mu.Unlock()

	saveRoom(room)
	logEvent(LogEntry{Event: "user_joined", RoomID: room.ID, User: userName})

	// Restore all active connections while preserving votes
	room.Mu.Lock()
	for name := range room.Users {
		if conn, ok := activeConnections.Load(room.ID + ":" + name); ok {
			room.Users[name].Conn = conn.(*websocket.Conn)
		}
	}
	room.Mu.Unlock()

	setupCloseHandler(conn, room, userName)
	broadcastRoomState(room)
	defer func() {
		activeConnections.Delete(room.ID + ":" + userName)
		handlePanic(conn, userName, room.ID)
	}()
	handleMessages(conn, room, userName)
}

func handleMessages(conn *websocket.Conn, room *Room, userName string) {
	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s in room %s: %v", userName, room.ID, err)
			}
			break
		}

		log.Printf("Received message from user %s in room %s: type=%s", userName, room.ID, msg.Type)

		switch msg.Type {
		case "vote":
			handleVote(room, userName, msg)
		case "reveal":
			handleReveal(room, userName)
		case "next":
			handleNext(room, userName)
		case "previous":
			handlePrevious(room, userName)
		default:
			log.Printf("Unknown message type from user %s in room %s: %s", userName, room.ID, msg.Type)
		}
	}
}

func broadcastRoomState(room *Room) {
	room.Mu.RLock()
	// Make a copy of users to avoid nil pointer
	users := make(map[string]*User)
	for name, user := range room.Users {
		users[name] = user
	}
	roomData := toRoomData(room)
	room.Mu.RUnlock()

	msg := Message{
		Type:    "roomState",
		Payload: roomData,
	}

	for _, user := range users {
		if user != nil && user.Conn != nil {
			if err := user.Conn.WriteJSON(msg); err != nil {
				log.Printf("Error broadcasting to user %s: %v", user.Name, err)
			}
		}
	}
}

func handleVote(room *Room, userName string, msg Message) {
	if room.GameMaster == userName {
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
		return
	}

	vote := int(voteFloat)
	currentRoom, err := getRoom(room.ID)
	if err != nil {
		log.Printf("Error getting room %s: %v", room.ID, err)
		return
	}

	currentRoom.Mu.Lock()
	for i, ticket := range currentRoom.Tickets {
		if ticket.ID == ticketID {
			currentRoom.Tickets[i].Votes[userName] = vote
			break
		}
	}
	currentRoom.Mu.Unlock()

	// Update connections while preserving votes
	currentRoom.Mu.Lock()
	for name := range currentRoom.Users {
		if conn, ok := activeConnections.Load(currentRoom.ID + ":" + name); ok {
			currentRoom.Users[name].Conn = conn.(*websocket.Conn)
		}
	}
	currentRoom.Mu.Unlock()

	saveRoom(currentRoom)
	broadcastRoomState(currentRoom)
}

func handleReveal(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	currentRoom, err := getRoom(room.ID)
	if err != nil {
		log.Printf("Error getting room %s: %v", room.ID, err)
		return
	}

	currentRoom.Mu.Lock()
	for name := range currentRoom.Users {
		if conn, ok := activeConnections.Load(currentRoom.ID + ":" + name); ok {
			currentRoom.Users[name].Conn = conn.(*websocket.Conn)
		}
	}
	currentRoom.VotesRevealed = true
	currentRoom.Mu.Unlock()

	saveRoom(currentRoom)
	broadcastRoomState(currentRoom)
}

func handleNext(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	currentRoom, err := getRoom(room.ID)
	if err != nil {
		log.Printf("Error getting room %s: %v", room.ID, err)
		return
	}

	currentRoom.Mu.Lock()
	if currentRoom.CurrentTicket < len(currentRoom.Tickets)-1 {
		currentRoom.CurrentTicket++
		currentRoom.VotesRevealed = false
	}
	for name := range currentRoom.Users {
		if conn, ok := activeConnections.Load(currentRoom.ID + ":" + name); ok {
			currentRoom.Users[name].Conn = conn.(*websocket.Conn)
		}
	}
	currentRoom.Mu.Unlock()

	saveRoom(currentRoom)
	broadcastRoomState(currentRoom)
}

func handlePrevious(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	currentRoom, err := getRoom(room.ID)
	if err != nil {
		log.Printf("Error getting room %s: %v", room.ID, err)
		return
	}

	currentRoom.Mu.Lock()
	if currentRoom.CurrentTicket > 0 {
		currentRoom.CurrentTicket--
		currentRoom.VotesRevealed = false
	}
	for name := range currentRoom.Users {
		if conn, ok := activeConnections.Load(currentRoom.ID + ":" + name); ok {
			currentRoom.Users[name].Conn = conn.(*websocket.Conn)
		}
	}
	currentRoom.Mu.Unlock()

	saveRoom(currentRoom)
	broadcastRoomState(currentRoom)
}
