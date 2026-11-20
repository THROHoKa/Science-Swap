package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"time"
)

type versionResponse struct {
	Version string `json:"version"`
}

func routes(db *sql.DB, static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.FileServerFS(static))
	mux.Handle("GET /assets/", http.FileServerFS(static))

	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var response versionResponse
		if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&response.Version); err != nil {
			log.Printf("Read PostgreSQL version: %v", err)
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Write version response: %v", err)
		}
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}
