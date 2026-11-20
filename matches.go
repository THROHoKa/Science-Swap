package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const reciprocalRegions = `
	EXISTS (
		SELECT 1 FROM "SSwap_preferences" f WHERE f.account_id = me.account_id
		  AND sswap_region_contains(f.scope, f.value, i.country_code, i.continent_code, i.subdivision_code)
	)
	AND EXISTS (
		SELECT 1 FROM "SSwap_preferences" f WHERE f.account_id = p.account_id
		  AND sswap_region_contains(f.scope, f.value, home.country_code, home.continent_code, home.subdivision_code)
	)`

func (a api) getMatches(w http.ResponseWriter, r *http.Request, id string) {
	query := r.URL.Query()
	from, to := query.Get("from"), query.Get("to")
	minimum := 1
	if value := query.Get("minimum"); value != "" {
		var err error
		minimum, err = strconv.Atoi(value)
		if err != nil || (minimum != 1 && minimum != 3 && minimum != 6) {
			apiError(w, http.StatusBadRequest, "invalid_request")
			return
		}
	}
	scope, value := "world", ""
	switch query.Get("region") {
	case "", "world":
	case "europe":
		scope, value = "continent", "EU"
	case "eu":
		scope, value = "region_group", "EU"
	case "dach":
		scope, value = "region_group", "DACH"
	default:
		apiError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var savedFrom, savedTo string
	err := a.db.QueryRowContext(r.Context(), `
		SELECT to_char(lower(period), 'YYYY-MM'), to_char(upper(period)-1, 'YYYY-MM')
		FROM "SSwap_availabilities" WHERE account_id = $1::text::uuid`, id).Scan(&savedFrom, &savedTo)
	if errors.Is(err, sql.ErrNoRows) {
		apiError(w, http.StatusConflict, "profile_required")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if from == "" && to == "" {
		from, to = savedFrom, savedTo
	}
	start, end, valid := monthRange(from, to)
	if !valid {
		apiError(w, http.StatusBadRequest, "invalid_period")
		return
	}
	var data json.RawMessage
	err = a.db.QueryRowContext(r.Context(), `
		WITH requested AS (SELECT daterange($2::text::date, $3::text::date) AS period),
		results AS (
			SELECT p.account_id AS id, p.name, upper(overlap.period)-lower(overlap.period) AS days,
			jsonb_build_object(
				'id', p.account_id, 'title', p.title, 'name', p.name, 'field', p.isced_code,
				'teaching', p.teaching_areas, 'institution', `+institutionJSON+`, 'demo', account.is_demo,
				'from', to_char(lower(d.period), 'YYYY-MM'), 'to', to_char(upper(d.period)-1, 'YYYY-MM'),
				'overlapFrom', to_char(lower(overlap.period), 'YYYY-MM'),
				'overlapTo', to_char(upper(overlap.period)-1, 'YYYY-MM'),
				'preferences', (
					SELECT jsonb_agg(jsonb_build_object('scope', f.scope, 'value', f.value) ORDER BY f.scope, f.value)
					FROM "SSwap_preferences" f WHERE f.account_id = p.account_id
				),
				'sent', sent.from_id IS NOT NULL, 'received', received.from_id IS NOT NULL
			) AS item
			FROM "SSwap_profiles" me
			JOIN "SSwap_institutions" home ON home.id = me.institution_id
			CROSS JOIN requested
			JOIN "SSwap_availabilities" d ON d.period && requested.period AND d.account_id <> me.account_id
			JOIN "SSwap_profiles" p ON p.account_id = d.account_id AND p.institution_id <> me.institution_id
			JOIN "SSwap_accounts" account ON account.id = p.account_id
			JOIN "SSwap_institutions" i ON i.id = p.institution_id AND i.status = 'active'
			LEFT JOIN "SSwap_interests" sent ON sent.from_id = me.account_id AND sent.to_id = p.account_id
			LEFT JOIN "SSwap_interests" received ON received.from_id = p.account_id AND received.to_id = me.account_id
			CROSS JOIN LATERAL (SELECT d.period * requested.period AS period) overlap
			WHERE me.account_id = $1::text::uuid
			  AND ($4::boolean OR p.isced_code = me.isced_code)
			  AND sswap_region_contains($5, $6, i.country_code, i.continent_code, i.subdivision_code)
			  AND upper(overlap.period) >= lower(overlap.period) + make_interval(months => $7)
			  AND `+reciprocalRegions+`
			ORDER BY days DESC, p.name, p.account_id LIMIT 200
		)
		SELECT coalesce(jsonb_agg(item ORDER BY days DESC, name, id), '[]'::jsonb) FROM results`,
		id, start.Format(time.DateOnly), end.Format(time.DateOnly), query.Get("field") == "all", scope, value, minimum).Scan(&data)
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, struct {
		From    string          `json:"from"`
		To      string          `json:"to"`
		Results json.RawMessage `json:"results"`
	}{from, to, data})
}

func (a api) saveInterest(w http.ResponseWriter, r *http.Request, id string) {
	candidate := r.PathValue("candidate")
	var uuid pgtype.UUID
	if err := uuid.Scan(candidate); err != nil {
		apiError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var exists bool
	err := a.db.QueryRowContext(r.Context(), `
		WITH eligible AS (
			SELECT me.account_id AS from_id, p.account_id AS to_id
			FROM "SSwap_profiles" me
			JOIN "SSwap_institutions" home ON home.id = me.institution_id
			JOIN "SSwap_profiles" p ON p.account_id = $2::text::uuid
			JOIN "SSwap_institutions" i ON i.id = p.institution_id AND i.status = 'active'
			WHERE me.account_id = $1::text::uuid AND me.account_id <> p.account_id
			  AND p.institution_id <> me.institution_id AND `+reciprocalRegions+`
		), saved AS (
			INSERT INTO "SSwap_interests" (from_id, to_id)
			SELECT from_id, to_id FROM eligible ON CONFLICT DO NOTHING
		)
		SELECT EXISTS (SELECT 1 FROM eligible)`, id, candidate).Scan(&exists)
	if err != nil {
		internalError(w, err)
		return
	}
	if !exists {
		apiError(w, http.StatusNotFound, "match_unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
