package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func testDatabase(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("set TEST_DATABASE_URL to faculty_swap_test")
	}
	db, err := openDatabase(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var name string
	var versionNumber int
	err = db.QueryRowContext(ctx, "SELECT current_database(), current_setting('server_version_num')::int").Scan(&name, &versionNumber)
	if err != nil {
		t.Fatal(err)
	}
	if name != "faculty_swap_test" || versionNumber < 170000 {
		t.Fatalf("expected faculty_swap_test on PostgreSQL 17+, got %s on %d", name, versionNumber)
	}
	return db
}

func TestHelloWorld(t *testing.T) {
	db := testDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var extensionCount int
	err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM pg_extension
		WHERE extname IN ('pg_trgm', 'unaccent', 'btree_gist', 'citext', 'cube', 'earthdistance')
	`).Scan(&extensionCount)
	if err != nil {
		t.Fatal(err)
	}
	if extensionCount != 6 {
		t.Fatalf("expected all six extensions, got %d", extensionCount)
	}

	static, err := fs.Sub(files, "web/dist")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(db, static)
	for _, path := range []string{"/", "/login", "/profile", "/matches"} {
		page := httptest.NewRecorder()
		handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, path, nil))
		if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "/assets/") {
			t.Fatalf("%s: expected embedded frontend, got %d with body %q", path, page.Code, page.Body)
		}
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown path: expected HTTP 404, got %d", missing.Code)
	}

	version := httptest.NewRecorder()
	handler.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/api/version", nil))
	var payload versionResponse
	if err := json.Unmarshal(version.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected version JSON, got %d with body %q", version.Code, version.Body)
	}
	var expected string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&expected); err != nil {
		t.Fatal(err)
	}
	if version.Code != http.StatusOK || payload.Version != expected {
		t.Fatalf("expected HTTP 200 with PostgreSQL %q, got %d with %q", expected, version.Code, payload.Version)
	}
}
