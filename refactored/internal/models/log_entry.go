package models

import "time"

type LogEntry struct {
	Time     time.Time `json:"time"`
	Event    string    `json:"event"`
	RoomID   string    `json:"roomId"`
	User     string    `json:"user,omitempty"`
	TicketID string    `json:"ticketId,omitempty"`
	Vote     int       `json:"vote,omitempty"`
	Error    string    `json:"error,omitempty"`
}
