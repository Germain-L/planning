package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	activeConnections sync.Map // Store connections globally
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
