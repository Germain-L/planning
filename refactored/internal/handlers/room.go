package handlers

import (
	"encoding/json"
	"net/http"
	"planning/internal/services"
	"planning/internal/utils"
)

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var request struct {
		TicketIDs []string `json:"ticketIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.TicketIDs) == 0 {
		utils.WriteError(w, http.StatusBadRequest, "Invalid request body or no tickets provided")
		return
	}

	roomID, err := services.CreateRoom(request.TicketIDs)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Could not create room")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"roomId": roomID})
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
