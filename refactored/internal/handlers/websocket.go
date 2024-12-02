package handlers

import (
	"net/http"
	"planning/internal/services"
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomId")
	userName := r.URL.Query().Get("name")
	isGameMaster := r.URL.Query().Get("gamemaster") == "true"

	if err := services.HandleWebSocketConnection(w, r, roomID, userName, isGameMaster); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
