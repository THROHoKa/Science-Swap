package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"time"
)

type api struct{ db *sql.DB }

func (a api) register(mux *http.ServeMux) {
	protected := http.NewCrossOriginProtection()
	protected.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiError(w, http.StatusForbidden, "forbidden")
	}))
	handle := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, protected.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			handler(w, r.WithContext(ctx))
		})))
	}
	handle("POST /api/session", a.startSession)
	handle("DELETE /api/session", a.endSession)
	handle("GET /api/profile", a.requireAccount(a.getProfile))
	handle("PUT /api/profile", a.requireAccount(a.saveProfile))
	handle("GET /api/catalog", a.requireAccount(a.getCatalog))
	handle("GET /api/institutions", a.requireAccount(a.searchInstitutions))
	handle("GET /api/subdivisions", a.requireAccount(a.listSubdivisions))
	handle("GET /api/matches", a.requireAccount(a.getMatches))
	handle("GET /api/matches/{candidate}", a.requireAccount(a.getMatchProfile))
	handle("PUT /api/interests/{candidate}", a.requireAccount(a.saveInterest))
}

func (a api) requireAccount(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _, err := a.sessionAccount(r.Context(), a.db, r)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			apiError(w, http.StatusUnauthorized, "login_required")
		case err != nil:
			internalError(w, err)
		default:
			next(w, r, id)
		}
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		apiError(w, http.StatusUnsupportedMediaType, "invalid_request")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil || decoder.Decode(new(any)) != io.EOF {
		apiError(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

func respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write API response: %v", err)
	}
}

func apiError(w http.ResponseWriter, status int, code string) {
	respondJSON(w, status, map[string]string{"error": code})
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("API request failed: %v", err)
	apiError(w, http.StatusInternalServerError, "unavailable")
}
