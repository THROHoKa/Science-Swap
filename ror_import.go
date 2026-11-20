package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

type rorOrganization struct {
	ID      string
	Status  string
	Types   []string
	Domains []string
	Names   []struct {
		Value string
		Lang  string
		Types []string
	}
	Locations []struct {
		Details struct {
			City            string `json:"name"`
			Country         string `json:"country_code"`
			Continent       string `json:"continent_code"`
			Subdivision     string `json:"country_subdivision_code"`
			SubdivisionName string `json:"country_subdivision_name"`
			Lat             *float64
			Lng             *float64
		} `json:"geonames_details"`
	}
}

type rorImportResult struct {
	added, updated, deactivated int64
}

func importROR(ctx context.Context, conn *pgx.Conn, input io.Reader, release string) (rorImportResult, error) {
	var result rorImportResult
	decoder := json.NewDecoder(input)
	if token, err := decoder.Token(); err != nil || token != json.Delim('[') {
		return result, errors.New("expected a JSON array of ROR organizations")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, createRORStage); err != nil {
		return result, fmt.Errorf("create ROR staging table: %w", err)
	}
	columns := []string{
		"ror_id", "status", "name", "aliases", "acronyms", "labels", "types", "domains",
		"city", "subdivision_code", "subdivision_name", "country_code", "continent_code", "lat", "lng", "search_text",
	}
	var active int
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"ror_stage"}, columns, pgx.CopyFromFunc(func() ([]any, error) {
		for decoder.More() {
			var org rorOrganization
			if err := decoder.Decode(&org); err != nil {
				return nil, fmt.Errorf("decode ROR record: %w", err)
			}
			if !strings.HasPrefix(org.ID, "https://ror.org/") || org.ID == "https://ror.org/" {
				return nil, errors.New("record is missing a ROR ID")
			}
			switch org.Status {
			case "active":
				if !slices.Contains(org.Types, "education") {
					continue
				}
				active++
				return org.activeRow()
			case "inactive", "withdrawn":
				// Keep status-only rows so existing references are preserved.
				row := make([]any, len(columns))
				row[0], row[1] = org.ID, org.Status
				return row, nil
			default:
				return nil, fmt.Errorf("unknown ROR status %q for %s", org.Status, org.ID)
			}
		}
		if _, err := decoder.Token(); err != nil {
			return nil, fmt.Errorf("read end of ROR array: %w", err)
		}
		if _, err := decoder.Token(); err != io.EOF {
			return nil, errors.New("unexpected data after ROR array")
		}
		return nil, nil
	}))
	if err != nil {
		return result, fmt.Errorf("stage ROR organizations: %w", err)
	}
	if active == 0 {
		return result, errors.New("ROR archive contains no active educational institutions")
	}
	if _, err := tx.Exec(ctx, `LOCK TABLE "SSwap_institutions" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return result, fmt.Errorf("lock institutions for ROR import: %w", err)
	}
	if err := tx.QueryRow(ctx, upsertRORInstitutions, release).Scan(&result.added, &result.updated); err != nil {
		return result, fmt.Errorf("upsert ROR institutions: %w", err)
	}
	changed, err := tx.Exec(ctx, deactivateRORInstitutions, release)
	if err != nil {
		return result, fmt.Errorf("deactivate ROR institutions: %w", err)
	}
	result.deactivated = changed.RowsAffected()
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit ROR import: %w", err)
	}
	return result, nil
}

func (org rorOrganization) activeRow() ([]any, error) {
	if len(org.Locations) == 0 || len(org.Locations[0].Details.Country) != 2 {
		return nil, fmt.Errorf("missing country for ROR institution %s", org.ID)
	}
	location := org.Locations[0].Details
	var name string
	var aliases, acronyms, searchTerms []string
	labels := make(map[string]string)
	for _, entry := range org.Names {
		if slices.Contains(entry.Types, "ror_display") {
			name = entry.Value
		}
		if slices.Contains(entry.Types, "alias") {
			aliases = append(aliases, entry.Value)
		}
		if slices.Contains(entry.Types, "acronym") {
			acronyms = append(acronyms, entry.Value)
		}
		if entry.Lang != "" && slices.Contains(entry.Types, "label") {
			labels[entry.Lang] = entry.Value
		}
		searchTerms = append(searchTerms, entry.Value+" "+location.City)
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("missing display name for ROR institution %s", org.ID)
	}
	return []any{
		org.ID, org.Status, name, aliases, acronyms, labels, org.Types, org.Domains,
		location.City, location.Subdivision, location.SubdivisionName, location.Country,
		location.Continent, location.Lat, location.Lng, strings.Join(searchTerms, " "),
	}, nil
}

const createRORStage = `
CREATE TEMP TABLE ror_stage (
    ror_id text PRIMARY KEY,
    status text NOT NULL,
    name text,
    aliases text[],
    acronyms text[],
    labels jsonb,
    types text[],
    domains text[],
    city text,
    subdivision_code text,
    subdivision_name text,
    country_code char(2),
    continent_code char(2),
    lat double precision,
    lng double precision,
    search_text text
) ON COMMIT DROP`

const upsertRORInstitutions = `
WITH written AS (
    INSERT INTO "SSwap_institutions" AS i (
        ror_id, status, name, aliases, acronyms, labels, types, domains,
        city, subdivision_code, subdivision_name, country_code, continent_code,
        lat, lng, search_text, ror_release
    )
    SELECT ror_id, status, name, coalesce(aliases, '{}'), coalesce(acronyms, '{}'), labels,
        types, coalesce(domains, '{}'), nullif(city, ''), nullif(subdivision_code, ''),
        nullif(subdivision_name, ''), country_code, nullif(continent_code, ''), lat, lng,
        public.immutable_unaccent(lower(search_text)), $1
    FROM ror_stage
    WHERE status = 'active'
    ON CONFLICT (ror_id) DO UPDATE SET
        (status, name, aliases, acronyms, labels, types, domains, city,
         subdivision_code, subdivision_name, country_code, continent_code, lat, lng,
         search_text, ror_release, updated_at) =
        (excluded.status, excluded.name, excluded.aliases, excluded.acronyms, excluded.labels,
         excluded.types, excluded.domains, excluded.city, excluded.subdivision_code,
         excluded.subdivision_name, excluded.country_code, excluded.continent_code, excluded.lat,
         excluded.lng, excluded.search_text, excluded.ror_release, now())
    WHERE to_jsonb(i) - '{id,updated_at}'::text[] IS DISTINCT FROM
          to_jsonb(excluded) - '{id,updated_at}'::text[]
    RETURNING ror_id
)
SELECT count(*) FILTER (WHERE previous.ror_id IS NULL),
       count(*) FILTER (WHERE previous.ror_id IS NOT NULL)
FROM written LEFT JOIN "SSwap_institutions" previous USING (ror_id)`

const deactivateRORInstitutions = `
UPDATE "SSwap_institutions" i
SET status = unlisted.status, ror_release = $1, updated_at = now()
FROM (
    SELECT previous.ror_id, coalesce(stage.status, 'inactive') AS status
    FROM "SSwap_institutions" previous LEFT JOIN ror_stage stage USING (ror_id)
    WHERE stage.status IN ('inactive', 'withdrawn')
       OR (stage.ror_id IS NULL AND previous.status = 'active')
) unlisted
WHERE i.ror_id = unlisted.ror_id AND i.status IS DISTINCT FROM unlisted.status`
