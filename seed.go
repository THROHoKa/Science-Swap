package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func seedCommand(ctx context.Context, db *sql.DB, args []string) error {
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	count := flags.Int("count", 150, "number of sample profiles")
	randomSeed := flags.Int64("random-seed", 42, "seed for reproducible sample profiles")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *count < 1 || *count > 2000 {
		return errors.New("seed requires a count from 1 to 2000")
	}
	var database string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err != nil {
		return err
	}
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") ||
		(database != "faculty_swap_dev" && database != "faculty_swap_test") {
		return errors.New("sample profiles can only be seeded in development or test databases")
	}
	if err := seedProfiles(ctx, db, *count, *randomSeed); err != nil {
		return err
	}
	log.Printf("Seeded %d sample profiles", *count)
	return nil
}

func seedProfiles(ctx context.Context, db *sql.DB, count int, seed int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, coalesce(continent_code = 'EU', false) FROM "SSwap_institutions"
		WHERE status = 'active'
		ORDER BY ror_id NOT IN (
			'https://ror.org/03prydq77', 'https://ror.org/05f950310', 'https://ror.org/03dbr7087',
			'https://ror.org/05a28rw58', 'https://ror.org/046ak2485', 'https://ror.org/03hbmgt12',
			'https://ror.org/02kkvpp62', 'https://ror.org/05hy3tk52', 'https://ror.org/01nrxwf90',
			'https://ror.org/02e2c7k09', 'https://ror.org/01tgyzw49'
		), ror_id LIMIT 12`)
	if err != nil {
		return err
	}
	var european, other, all []int64
	for rows.Next() {
		var id int64
		var inEurope bool
		if err := rows.Scan(&id, &inEurope); err != nil {
			rows.Close()
			return err
		}
		all = append(all, id)
		if inEurope {
			european = append(european, id)
		} else {
			other = append(other, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(all) < 2 {
		return errors.New("import at least two ROR institutions before seeding profiles")
	}
	if len(european) == 0 {
		european = all
	}
	if len(other) == 0 {
		other = all
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(662573328)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM "SSwap_accounts"
		WHERE is_demo AND email::text ~ '^sample-[0-9]{4}@example\.org$'
		  AND substring(email::text FROM 8 FOR 4)::integer > $1`, count); err != nil {
		return err
	}

	first := []string{"Ada", "Alex", "Amira", "Arun", "Camille", "Elias", "Esra", "Hana", "Jonas", "Lea", "Lina", "Maya", "Noah", "Robin", "Sam"}
	last := []string{"Almeida", "Baumann", "Chen", "Dubois", "Fischer", "Garcia", "Hoffmann", "Ibrahim", "Jensen", "Kaur", "Martin", "Novak", "Park", "Santos", "Weber"}
	fields := []string{"061", "061", "061", "061", "054", "071", "031", "041", "091", "051"}
	rng := rand.New(rand.NewSource(seed))
	now := time.Now().UTC()
	year := now.Year()
	if now.Month() >= time.September {
		year++
	}
	academicYear := time.Date(year, time.September, 1, 0, 0, 0, 0, time.UTC)
	for i := range count {
		locations := european
		if i%10 >= 7 {
			locations = other
		}
		location := locations[rng.Intn(len(locations))]
		name := first[i%len(first)] + " " + last[(i/len(first)+i)%len(last)]
		start := academicYear.AddDate(0, []int{0, 1, 2, 6, -2}[rng.Intn(5)], 0)
		end := start.AddDate(0, 6+rng.Intn(13), 0)
		scope, value := "world", ""
		if i%10 < 7 {
			scope, value = "continent", "EU"
		}
		if _, err := tx.ExecContext(ctx, seedProfileSQL, fmt.Sprintf("sample-%04d@example.org", i+1),
			name, location, fields[i%len(fields)], start.Format(time.DateOnly), end.Format(time.DateOnly), scope, value); err != nil {
			return fmt.Errorf("seed profile %d: %w", i+1, err)
		}
	}
	return tx.Commit()
}

const seedProfileSQL = `
WITH a AS (
    INSERT INTO "SSwap_accounts" (email, is_demo) VALUES ($1, true)
    ON CONFLICT (email) WHERE is_demo DO UPDATE SET email = excluded.email RETURNING id
), p AS (
    INSERT INTO "SSwap_profiles" (account_id, name, institution_id, isced_code)
    SELECT id, $2, $3, $4 FROM a
    ON CONFLICT (account_id) DO UPDATE SET name = excluded.name, institution_id = excluded.institution_id,
      isced_code = excluded.isced_code RETURNING account_id
), d AS (
    INSERT INTO "SSwap_availabilities" SELECT account_id, daterange($5::text::date, $6::text::date) FROM p
    ON CONFLICT (account_id) DO UPDATE SET period = excluded.period
)
INSERT INTO "SSwap_preferences" SELECT account_id, $7, $8 FROM p ON CONFLICT DO NOTHING`
