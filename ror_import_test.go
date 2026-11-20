package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

func TestRORImport(t *testing.T) {
	db := testDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	checkReferenceData(t, ctx, db)

	fixture, err := os.ReadFile("testdata/ror-sample.json")
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	err = connection.Raw(func(driverConn any) error {
		conn := driverConn.(*stdlib.Conn).Conn()
		_, err := conn.Exec(ctx, `CREATE TEMP TABLE "SSwap_institutions"
			(LIKE public."SSwap_institutions" INCLUDING ALL)`)
		if err != nil {
			return err
		}
		defer conn.Exec(ctx, `DROP TABLE pg_temp."SSwap_institutions"`)

		run := func(data []byte, release string, want rorImportResult) {
			t.Helper()
			got, err := importROR(ctx, conn, bytes.NewReader(data), release)
			if err != nil || got != want {
				t.Errorf("import %s: got %+v (%v), want %+v", release, got, err, want)
			}
		}
		run(fixture, "v2.1", rorImportResult{added: 3})
		run(fixture, "v2.1", rorImportResult{})

		var id int64
		var label string
		err = conn.QueryRow(ctx, `
			SELECT id, labels->>'en' FROM "SSwap_institutions"
			WHERE ror_id = 'https://ror.org/test-berlin' AND status = 'active'
			  AND public.immutable_unaccent(lower('FU Berlin')) <% search_text
			  AND search_text LIKE '%universitat%'`).Scan(&id, &label)
		if err != nil {
			return err
		}
		if label != "Free University of Berlin" {
			t.Errorf("unexpected English label %q", label)
		}

		var records []json.RawMessage
		if err := json.Unmarshal(fixture, &records); err != nil {
			return err
		}
		next, err := json.Marshal([]json.RawMessage{
			[]byte(strings.Replace(string(records[0]), "Freie Universität Berlin", "Freie Universität", 1)),
			[]byte(`{"id":"https://ror.org/test-rosenheim","status":"withdrawn","types":["education"]}`),
		})
		if err != nil {
			return err
		}
		run(next, "v2.2", rorImportResult{updated: 1, deactivated: 2})
		if _, err := importROR(ctx, conn, bytes.NewReader(next[:len(next)-1]), "v2.3"); err == nil {
			t.Error("malformed snapshot was accepted")
		}
		if _, err := importROR(ctx, conn, strings.NewReader("[]"), "v2.3"); err == nil {
			t.Error("empty snapshot was accepted")
		}

		var afterID int64
		var name, release, rosenheim, linz string
		err = conn.QueryRow(ctx, `
			SELECT berlin.id, berlin.name, berlin.ror_release, rosenheim.status, linz.status
			FROM "SSwap_institutions" berlin, "SSwap_institutions" rosenheim, "SSwap_institutions" linz
			WHERE berlin.ror_id = 'https://ror.org/test-berlin'
			  AND rosenheim.ror_id = 'https://ror.org/test-rosenheim'
			  AND linz.ror_id = 'https://ror.org/test-linz'`).Scan(&afterID, &name, &release, &rosenheim, &linz)
		if err != nil {
			return err
		}
		if id != afterID || name != "Freie Universität" || release != "v2.2" || rosenheim != "withdrawn" || linz != "inactive" {
			t.Errorf("unexpected post-import state: %d %q %q %q %q", afterID, name, release, rosenheim, linz)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkReferenceData(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var fields, euCountries, dachCountries int
	err := db.QueryRowContext(ctx, `
		SELECT (SELECT count(*) FROM "SSwap_isced_fields" WHERE name_de <> '' AND name_en <> ''),
		       (SELECT cardinality(country_codes) FROM "SSwap_region_groups" WHERE code = 'EU'),
		       (SELECT cardinality(country_codes) FROM "SSwap_region_groups" WHERE code = 'DACH')`).
		Scan(&fields, &euCountries, &dachCountries)
	if err != nil {
		t.Fatal(err)
	}
	if fields != 25 || euCountries != 27 || dachCountries != 3 {
		t.Fatalf("unexpected seed totals: %d fields, %d EU countries, %d DACH countries", fields, euCountries, dachCountries)
	}
}
