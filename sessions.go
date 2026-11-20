package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
)

const (
	sessionCookie = "science_swap_session"
	sessionTTL    = 30 * 24 * time.Hour
)

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (a api) sessionAccount(ctx context.Context, db rowQueryer, r *http.Request) (id, email string, err error) {
	hash := requestSessionHash(r)
	if hash == nil {
		return "", "", sql.ErrNoRows
	}
	err = db.QueryRowContext(ctx, `
		SELECT a.id::text, a.email::text
		FROM "SSwap_sessions" s JOIN "SSwap_accounts" a ON a.id = s.account_id
		WHERE s.token_hash = $1 AND s.expires_at > now() AND NOT a.is_demo`, hash).Scan(&id, &email)
	return id, email, err
}

func requestSessionHash(r *http.Request) []byte {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	key, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(key) != 32 {
		return nil
	}
	hash := sha256.Sum256(key)
	return hash[:]
}

func (a api) startSession(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &input, 16<<10) {
		return
	}
	email := strings.TrimSpace(input.Email)
	address, err := mail.ParseAddress(email)
	if err != nil || len(email) > 254 || address.Address != email {
		apiError(w, http.StatusBadRequest, "invalid_email")
		return
	}
	ctx := r.Context()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback()

	// An active session can continue its own account. An email alone proves no ownership.
	id, currentEmail, err := a.sessionAccount(ctx, tx, r)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		internalError(w, err)
		return
	}
	if id == "" || !strings.EqualFold(email, currentEmail) {
		if err := tx.QueryRowContext(ctx, `INSERT INTO "SSwap_accounts" (email) VALUES ($1) RETURNING id::text`, email).Scan(&id); err != nil {
			internalError(w, err)
			return
		}
	}
	var key [32]byte
	rand.Read(key[:])
	hash := sha256.Sum256(key[:])
	expires := time.Now().Add(sessionTTL)
	if _, err := tx.ExecContext(ctx, `DELETE FROM "SSwap_sessions" WHERE expires_at <= now() OR token_hash = $1`, requestSessionHash(r)); err != nil {
		internalError(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO "SSwap_sessions" VALUES ($1, $2::text::uuid, $3)`, hash[:], id, expires); err != nil {
		internalError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, err)
		return
	}
	setSessionCookie(w, r, base64.RawURLEncoding.EncodeToString(key[:]), expires, int(sessionTTL.Seconds()))
	w.WriteHeader(http.StatusNoContent)
}

func (a api) endSession(w http.ResponseWriter, r *http.Request) {
	if hash := requestSessionHash(r); hash != nil {
		if _, err := a.db.ExecContext(r.Context(), `DELETE FROM "SSwap_sessions" WHERE token_hash = $1`, hash); err != nil {
			internalError(w, err)
			return
		}
	}
	setSessionCookie(w, r, "", time.Unix(1, 0), -1)
	w.WriteHeader(http.StatusNoContent)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expires time.Time, age int) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || os.Getenv("FLY_APP_NAME") != "", Expires: expires, MaxAge: age,
	})
}
