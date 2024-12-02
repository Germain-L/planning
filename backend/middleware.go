package main

import (
	"net/http"
	"os"
)

func corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://planning.germainleignel.com")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func adminMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key != os.Getenv("ADMIN_KEY") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}
