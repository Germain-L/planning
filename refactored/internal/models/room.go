package models

import "sync"

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
