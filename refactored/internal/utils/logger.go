package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func InitLogger() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Llongfile)
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}
