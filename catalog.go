package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"
)

const institutionJSON = `CASE WHEN i.id IS NOT NULL THEN jsonb_build_object(
	'id', i.id, 'name', i.name, 'city', coalesce(i.city, ''),
	'country', i.country_code, 'continent', coalesce(i.continent_code::text, ''),
	'subdivision', coalesce(i.subdivision_name, '')
) END`

func (a api) getCatalog(w http.ResponseWriter, r *http.Request, _ string) {
	var data json.RawMessage
	err := a.db.QueryRowContext(r.Context(), `
		SELECT jsonb_build_object(
			'fields', coalesce((SELECT jsonb_agg(jsonb_build_object('code', code, 'en', name_en, 'de', name_de) ORDER BY code)
				FROM "SSwap_isced_fields"), '[]'::jsonb),
			'regionGroups', coalesce((SELECT jsonb_agg(jsonb_build_object('code', code, 'en', name_en, 'de', name_de, 'countries', country_codes) ORDER BY code)
				FROM "SSwap_region_groups"), '[]'::jsonb),
			'countries', coalesce((SELECT jsonb_agg(DISTINCT country_code ORDER BY country_code)
				FROM "SSwap_institutions" WHERE status = 'active'), '[]'::jsonb)
		)`).Scan(&data)
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (a api) searchInstitutions(w http.ResponseWriter, r *http.Request, _ string) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) < 2 {
		respondJSON(w, http.StatusOK, []any{})
		return
	}
	if utf8.RuneCountInString(query) > 80 || strings.ContainsRune(query, 0) {
		apiError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var data json.RawMessage
	err := a.db.QueryRowContext(r.Context(), `
		WITH term AS (SELECT public.immutable_unaccent(lower($1::text)) AS q)
		SELECT coalesce(jsonb_agg(result.item ORDER BY result.score DESC, result.name, result.id), '[]'::jsonb)
		FROM (
			SELECT `+institutionJSON+` AS item, word_similarity(term.q, i.search_text) AS score, i.name, i.id
			FROM "SSwap_institutions" i CROSS JOIN term
			WHERE i.status = 'active' AND term.q <% i.search_text
			ORDER BY score DESC, i.name, i.id LIMIT 10
		) AS result`, query).Scan(&data)
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func (a api) listSubdivisions(w http.ResponseWriter, r *http.Request, _ string) {
	country := r.URL.Query().Get("country")
	if len(country) != 2 || country[0] < 'A' || country[0] > 'Z' || country[1] < 'A' || country[1] > 'Z' {
		apiError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var data json.RawMessage
	err := a.db.QueryRowContext(r.Context(), `
		SELECT coalesce(jsonb_agg(jsonb_build_object('code', code, 'name', name) ORDER BY name, code), '[]'::jsonb)
		FROM (
			SELECT subdivision_code AS code, min(subdivision_name) AS name
			FROM "SSwap_institutions"
			WHERE country_code = $1 AND status = 'active' AND coalesce(subdivision_name, '') <> '' AND coalesce(subdivision_code, '') <> ''
			GROUP BY subdivision_code
		) AS subdivisions`, country).Scan(&data)
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}
