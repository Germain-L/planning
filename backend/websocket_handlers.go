package main

import (
	"log"

	"github.com/gorilla/websocket"
)

func setupWebSocketConnection(conn *websocket.Conn, room *Room, userName string, isGameMaster bool) {
	setupCloseHandler(conn, room, userName)
	user := &User{Name: userName, Conn: conn} // Create user with connection

	room.Mu.Lock()
	if isGameMaster && room.GameMaster == "" {
		room.GameMaster = userName
		log.Printf("Set game master %s for room %s", userName, room.ID)
	}
	room.Users[userName] = user // Store user with active connection
	room.Mu.Unlock()

	saveRoom(room)
	logEvent(LogEntry{Event: "user_joined", RoomID: room.ID, User: userName})

	defer handlePanic(conn, userName, room.ID)
	broadcastRoomState(room)

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
		default:
			log.Printf("Unknown message type from user %s in room %s: %s", userName, room.ID, msg.Type)
		}
	}
}

func broadcastRoomState(room *Room) {
	room.Mu.RLock()
	roomData := toRoomData(room)
	room.Mu.RUnlock()

	msg := Message{
		Type:    "roomState",
		Payload: roomData,
	}

	for _, user := range room.Users {
		if user.Conn != nil {
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

	saveRoom(room)
	broadcastRoomState(room)
}

func handleReveal(room *Room, userName string) {
	if room.GameMaster != userName {
		return
	}

	room.Mu.Lock()
	room.VotesRevealed = true
	room.Mu.Unlock()

	saveRoom(room)
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

	saveRoom(room)
	broadcastRoomState(room)
}
