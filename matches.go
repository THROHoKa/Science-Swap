package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func (a api) getMatches(w http.ResponseWriter, r *http.Request, id string) {
	query := r.URL.Query()
	from, to := query.Get("from"), query.Get("to")
	fields := query["field"]
	switch {
	case len(fields) == 1 && fields[0] == "all":
		fields = []string{}
	case len(fields) == 1 && fields[0] == "same":
		fields = nil
	default:
		for _, code := range fields {
			if len(code) != 3 || strings.Trim(code, "0123456789") != "" {
				apiError(w, http.StatusBadRequest, "invalid_request")
				return
			}
		}
	}
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
	region := query.Get("region")
	switch region {
	case "", "world":
	case "europe":
		scope, value = "continent", "EU"
	case "AF", "AN", "AS", "NA", "OC", "SA":
		scope, value = "continent", region
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
		SELECT coalesce(to_char(lower(d.period), 'YYYY-MM'), ''), coalesce(to_char(upper(d.period)-1, 'YYYY-MM'), '')
		FROM "SSwap_profiles" p LEFT JOIN "SSwap_availabilities" d USING (account_id)
		WHERE p.account_id = $1::text::uuid`, id).Scan(&savedFrom, &savedTo)
	if errors.Is(err, sql.ErrNoRows) {
		apiError(w, http.StatusConflict, "profile_required")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if !query.Has("from") && !query.Has("to") {
		from, to = savedFrom, savedTo
	}
	start, end, valid := optionalMonthRange(from, to)
	if !valid {
		apiError(w, http.StatusBadRequest, "invalid_period")
		return
	}
	var data json.RawMessage
	err = a.db.QueryRowContext(r.Context(), `
		WITH requested AS (
			SELECT CASE WHEN $2::text <> '' THEN daterange($2::text::date, $3::text::date) END AS period
		),
		results AS (
			SELECT p.account_id AS id, p.name, upper(overlap.period)-lower(overlap.period) AS days,
			jsonb_build_object(
				'id', p.account_id, 'title', p.title, 'name', p.name, 'field', p.isced_code,
				'institution', `+institutionJSON+`, 'demo', account.is_demo,
				'from', coalesce(to_char(lower(d.period), 'YYYY-MM'), ''),
				'to', coalesce(to_char(upper(d.period)-1, 'YYYY-MM'), ''),
				'overlapFrom', coalesce(to_char(lower(overlap.period), 'YYYY-MM'), ''),
				'overlapTo', coalesce(to_char(upper(overlap.period)-1, 'YYYY-MM'), ''),
				'preferences', `+profilePreferencesJSON+`,
				'sent', sent.from_id IS NOT NULL, 'received', received.from_id IS NOT NULL
			) AS item
			FROM "SSwap_profiles" me
			CROSS JOIN requested
			JOIN "SSwap_profiles" p ON p.account_id <> me.account_id AND p.institution_id <> me.institution_id
			LEFT JOIN "SSwap_availabilities" d ON d.account_id = p.account_id
			JOIN "SSwap_accounts" account ON account.id = p.account_id
			JOIN "SSwap_institutions" i ON i.id = p.institution_id AND i.status = 'active'
			LEFT JOIN "SSwap_interests" sent ON sent.from_id = me.account_id AND sent.to_id = p.account_id
			LEFT JOIN "SSwap_interests" received ON received.from_id = p.account_id AND received.to_id = me.account_id
			CROSS JOIN LATERAL (
				SELECT CASE
					WHEN d.period IS NULL THEN requested.period
					WHEN requested.period IS NULL THEN d.period
					ELSE d.period * requested.period
				END AS period
			) overlap
			WHERE me.account_id = $1::text::uuid
			  AND CASE
				WHEN $4::text[] IS NULL THEN p.isced_code = me.isced_code
				WHEN cardinality($4::text[]) = 0 THEN true
				ELSE p.isced_code::text = ANY($4::text[])
			  END
			  AND sswap_region_contains($5, $6, i.country_code, i.continent_code, i.subdivision_code)
			  AND (requested.period IS NULL OR (
				(d.period IS NULL OR d.period && requested.period)
				AND upper(overlap.period) >= lower(overlap.period) + make_interval(months => $7)
			  ))
			ORDER BY days DESC NULLS LAST, p.name, p.account_id LIMIT 200
		)
		SELECT coalesce(jsonb_agg(item ORDER BY days DESC NULLS LAST, name, id), '[]'::jsonb) FROM results`,
		id, start, end, fields, scope, value, minimum).Scan(&data)
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

func (a api) getMatchProfile(w http.ResponseWriter, r *http.Request, id string) {
	candidate := r.PathValue("candidate")
	var uuid pgtype.UUID
	if err := uuid.Scan(candidate); err != nil {
		apiError(w, http.StatusNotFound, "match_unavailable")
		return
	}
	var data json.RawMessage
	err := a.db.QueryRowContext(r.Context(), `
		SELECT jsonb_build_object(
			'id', p.account_id, 'name', p.name, 'title', p.title, 'field', p.isced_code,
			'institution', `+institutionJSON+`, 'demo', account.is_demo,
			'teachingAreas', `+teachingAreasJSON+`,
			'from', coalesce(to_char(lower(d.period), 'YYYY-MM'), ''),
			'to', coalesce(to_char(upper(d.period)-1, 'YYYY-MM'), ''),
			'preferences', `+profilePreferencesJSON+`,
			'sent', sent.from_id IS NOT NULL, 'received', received.from_id IS NOT NULL
		)
		FROM "SSwap_profiles" me
		JOIN "SSwap_profiles" p ON p.account_id = $2::text::uuid
		JOIN "SSwap_institutions" i ON i.id = p.institution_id AND i.status = 'active'
		JOIN "SSwap_accounts" account ON account.id = p.account_id
		LEFT JOIN "SSwap_availabilities" d ON d.account_id = p.account_id
		LEFT JOIN "SSwap_interests" sent ON sent.from_id = me.account_id AND sent.to_id = p.account_id
		LEFT JOIN "SSwap_interests" received ON received.from_id = p.account_id AND received.to_id = me.account_id
		WHERE me.account_id = $1::text::uuid AND p.institution_id <> me.institution_id`, id, candidate).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		apiError(w, http.StatusNotFound, "match_unavailable")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, data)
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
			JOIN "SSwap_profiles" p ON p.account_id = $2::text::uuid
			JOIN "SSwap_institutions" i ON i.id = p.institution_id AND i.status = 'active'
			WHERE me.account_id = $1::text::uuid AND me.account_id <> p.account_id
			  AND p.institution_id <> me.institution_id
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
