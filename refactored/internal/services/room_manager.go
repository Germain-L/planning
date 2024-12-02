package services

import (
	"sync"

	"planning/internal/models"

	"github.com/google/uuid"
)

var (
	rooms   = make(map[string]*models.Room)
	roomsMu sync.RWMutex
)

func CreateRoom(ticketIDs []string) (string, error) {
	roomID := uuid.New().String()
	tickets := make([]models.Ticket, len(ticketIDs))
	for i, id := range ticketIDs {
		tickets[i] = models.Ticket{
			ID:    id,
			Votes: make(map[string]int),
		}
	}

	room := &models.Room{
		ID:            roomID,
		Tickets:       tickets,
		Users:         make(map[string]*models.User),
		CurrentTicket: 0,
		VotesRevealed: false,
	}

	roomsMu.Lock()
	defer roomsMu.Unlock()
	rooms[roomID] = room

	return roomID, nil
}
