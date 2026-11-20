package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type preference struct {
	Scope string `json:"scope"`
	Value string `json:"value"`
}

type teachingArea struct {
	Title      string  `json:"title"`
	Experience *string `json:"experience"`
}

type profileInput struct {
	Title         string         `json:"title"`
	Name          string         `json:"name"`
	Institution   int64          `json:"institutionId"`
	Field         string         `json:"field"`
	TeachingAreas []teachingArea `json:"teachingAreas"`
	From          string         `json:"from"`
	To            string         `json:"to"`
	Preferences   []preference   `json:"preferences"`
}

// Profiles saved before structured teaching areas retain their text without an assigned experience.
const teachingAreasJSON = `coalesce(p.teaching_entries, CASE
	WHEN btrim(p.teaching_areas) <> '' THEN jsonb_build_array(
		jsonb_build_object('title', btrim(p.teaching_areas), 'experience', NULL))
	ELSE '[]'::jsonb END)`

const profilePreferencesJSON = `coalesce((
	SELECT jsonb_agg(jsonb_build_object('scope', f.scope, 'value', f.value) ORDER BY f.scope, f.value)
	FROM "SSwap_preferences" f WHERE f.account_id = p.account_id
), '[]'::jsonb)`

func (a api) getProfile(w http.ResponseWriter, r *http.Request, id string) {
	var data json.RawMessage
	err := a.db.QueryRowContext(r.Context(), `
		SELECT jsonb_build_object(
			'email', a.email, 'title', coalesce(p.title, ''), 'name', coalesce(p.name, ''),
			'institution', `+institutionJSON+`, 'field', coalesce(p.isced_code::text, ''),
			'teachingAreas', `+teachingAreasJSON+`,
			'from', coalesce(to_char(lower(d.period), 'YYYY-MM'), ''),
			'to', coalesce(to_char(upper(d.period) - 1, 'YYYY-MM'), ''),
			'preferences', `+profilePreferencesJSON+`
		)
		FROM "SSwap_accounts" a
		LEFT JOIN "SSwap_profiles" p ON p.account_id = a.id
		LEFT JOIN "SSwap_institutions" i ON i.id = p.institution_id
		LEFT JOIN "SSwap_availabilities" d ON d.account_id = a.id
		WHERE a.id = $1::text::uuid`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		apiError(w, http.StatusUnauthorized, "login_required")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, data)
}

func optionalMonthRange(from, to string) (string, string, bool) {
	if from == "" && to == "" {
		return "", "", true
	}
	start, startErr := time.Parse("2006-01", from)
	end, endErr := time.Parse("2006-01", to)
	if startErr != nil || endErr != nil || start.Year() < 1 || end.Year() >= 9999 || end.Before(start) {
		return "", "", false
	}
	return start.Format(time.DateOnly), end.AddDate(0, 1, 0).Format(time.DateOnly), true
}

func (a api) saveProfile(w http.ResponseWriter, r *http.Request, id string) {
	var p profileInput
	if !decodeJSON(w, r, &p, 72<<10) {
		return
	}
	p.Title, p.Name = strings.TrimSpace(p.Title), strings.TrimSpace(p.Name)
	if p.Name == "" || utf8.RuneCountInString(p.Name) > 120 || utf8.RuneCountInString(p.Title) > 40 ||
		p.Institution <= 0 || len(p.Field) != 3 || strings.ContainsRune(p.Title+p.Name, 0) {
		apiError(w, http.StatusBadRequest, "invalid_profile")
		return
	}
	start, end, valid := optionalMonthRange(p.From, p.To)
	if !valid {
		apiError(w, http.StatusBadRequest, "invalid_period")
		return
	}
	if len(p.Preferences) > 30 {
		apiError(w, http.StatusBadRequest, "invalid_preferences")
		return
	}
	if p.TeachingAreas == nil {
		p.TeachingAreas = []teachingArea{}
	}
	if len(p.TeachingAreas) > 20 {
		apiError(w, http.StatusBadRequest, "invalid_teaching_areas")
		return
	}
	var legacyTitle string
	for i := range p.TeachingAreas {
		area := &p.TeachingAreas[i]
		area.Title = strings.TrimSpace(area.Title)
		if area.Title == "" || utf8.RuneCountInString(area.Title) > 500 || strings.ContainsRune(area.Title, 0) {
			apiError(w, http.StatusBadRequest, "invalid_teaching_areas")
			return
		}
		if area.Experience == nil {
			if legacyTitle != "" {
				apiError(w, http.StatusBadRequest, "invalid_teaching_areas")
				return
			}
			legacyTitle = area.Title
			continue
		}
		switch *area.Experience {
		case "full_lecture", "exercise_only", "single_units", "general_knowledge":
		default:
			apiError(w, http.StatusBadRequest, "invalid_teaching_areas")
			return
		}
	}
	for _, pref := range p.Preferences {
		if strings.ContainsRune(pref.Scope+pref.Value, 0) {
			apiError(w, http.StatusBadRequest, "invalid_preferences")
			return
		}
	}
	ctx := r.Context()
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, err)
		return
	}
	defer tx.Rollback()

	// Serialize competing saves of the same profile and its related rows.
	var locked string
	if err := tx.QueryRowContext(ctx, `SELECT id::text FROM "SSwap_accounts" WHERE id = $1::text::uuid FOR UPDATE`, id).Scan(&locked); err != nil {
		internalError(w, err)
		return
	}
	if legacyTitle != "" {
		var previouslySaved bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM "SSwap_profiles" p WHERE p.account_id = $1::text::uuid
				AND `+teachingAreasJSON+` @> jsonb_build_array(jsonb_build_object('title', $2::text, 'experience', NULL)))`,
			id, legacyTitle).Scan(&previouslySaved)
		if err != nil {
			internalError(w, err)
			return
		}
		if !previouslySaved {
			apiError(w, http.StatusBadRequest, "invalid_teaching_areas")
			return
		}
	}
	teaching, _ := json.Marshal(p.TeachingAreas)
	saved, err := tx.ExecContext(ctx, `
		INSERT INTO "SSwap_profiles" (account_id, title, name, institution_id, isced_code, teaching_entries)
		SELECT $1::text::uuid, $2, $3, i.id, f.code, $6::text::jsonb
		FROM "SSwap_institutions" i CROSS JOIN "SSwap_isced_fields" f
		WHERE i.id = $4 AND i.status = 'active' AND f.code = $5
		ON CONFLICT (account_id) DO UPDATE SET title = excluded.title, name = excluded.name,
			institution_id = excluded.institution_id, isced_code = excluded.isced_code, teaching_entries = excluded.teaching_entries`,
		id, p.Title, p.Name, p.Institution, p.Field, string(teaching))
	if err != nil {
		internalError(w, err)
		return
	}
	if count, _ := saved.RowsAffected(); count == 0 {
		apiError(w, http.StatusBadRequest, "invalid_reference")
		return
	}
	if start == "" {
		_, err = tx.ExecContext(ctx, `DELETE FROM "SSwap_availabilities" WHERE account_id = $1::text::uuid`, id)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO "SSwap_availabilities" VALUES ($1::text::uuid, daterange($2::text::date, $3::text::date))
			ON CONFLICT (account_id) DO UPDATE SET period = excluded.period`, id, start, end)
	}
	if err != nil {
		internalError(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM "SSwap_preferences" WHERE account_id = $1::text::uuid`, id); err != nil {
		internalError(w, err)
		return
	}
	if p.Preferences == nil {
		p.Preferences = []preference{}
	}
	payload, _ := json.Marshal(p.Preferences)
	saved, err = tx.ExecContext(ctx, `
		INSERT INTO "SSwap_preferences" (account_id, scope, value)
		SELECT $1::text::uuid, scope, value FROM jsonb_to_recordset($2::text::jsonb) AS f(scope text, value text)
		WHERE (scope = 'world' AND value = '')
		   OR (scope = 'continent' AND value IN ('AF', 'AN', 'AS', 'EU', 'NA', 'OC', 'SA'))
		   OR (scope = 'region_group' AND EXISTS (SELECT 1 FROM "SSwap_region_groups" WHERE code = value))
		   OR (scope = 'country' AND EXISTS (SELECT 1 FROM "SSwap_institutions" WHERE country_code::text = value AND status = 'active'))
		   OR (scope = 'subdivision' AND EXISTS (
			   SELECT 1 FROM "SSwap_institutions" WHERE country_code::text = split_part(value, ':', 1)
			     AND subdivision_code = split_part(value, ':', 2)
			     AND country_code || ':' || subdivision_code = value AND status = 'active'
		   ))
		ON CONFLICT DO NOTHING`, id, string(payload))
	if err != nil {
		internalError(w, err)
		return
	}
	if count, _ := saved.RowsAffected(); count != int64(len(p.Preferences)) {
		apiError(w, http.StatusBadRequest, "invalid_preferences")
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
