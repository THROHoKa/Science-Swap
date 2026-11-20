package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestExchangeWorkflow(t *testing.T) {
	db := testDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	tag := strings.ToLower(rand.Text())
	email := "test-" + tag + "@example.org"
	var institutions [3]int64
	for index, location := range []struct{ country, continent string }{{"DE", "EU"}, {"NL", "EU"}, {"US", "NA"}} {
		country := location.country
		err := db.QueryRowContext(ctx, `
			INSERT INTO "SSwap_institutions" (ror_id,name,types,status,country_code,continent_code,search_text,ror_release)
			VALUES ($1,$2,'{education}','active',$3,$4,$5,'test') RETURNING id`,
			"https://ror.org/test-"+tag+"-"+country, "Test campus "+country, country, location.continent, tag+" "+country).Scan(&institutions[index])
		if err != nil {
			t.Fatal(err)
		}
		id := institutions[index]
		t.Cleanup(func() { db.Exec(`DELETE FROM "SSwap_institutions" WHERE id = $1`, id) })
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM "SSwap_accounts" WHERE email = $1`, email) })

	static, err := fs.Sub(files, "web/dist")
	if err != nil {
		t.Fatal(err)
	}
	handler := routes(db, static)
	request := func(method, path string, body any, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, "https://science-swap.example"+path, strings.NewReader(string(payload)))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d %s, want %d", method, path, w.Code, w.Body, status)
		}
		return w
	}
	login := func(previous *http.Cookie) *http.Cookie {
		t.Helper()
		w := request("POST", "/api/session", map[string]string{"email": email}, previous, http.StatusNoContent)
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].Name != sessionCookie {
			t.Fatal("expected an HttpOnly, Secure session cookie")
		}
		return cookies[0]
	}
	readProfile := func(cookie *http.Cookie) map[string]any {
		t.Helper()
		var p map[string]any
		w := request("GET", "/api/profile", nil, cookie, http.StatusOK)
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	request("GET", "/api/profile", nil, nil, http.StatusUnauthorized)
	request("POST", "/api/session", map[string]string{"email": "broken"}, nil, http.StatusBadRequest)
	crossSite := httptest.NewRequest("POST", "https://science-swap.example/api/session", strings.NewReader("{}"))
	crossSite.Header.Set("Origin", "https://somewhere.example")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, crossSite)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("cross-origin login received status %d", blocked.Code)
	}

	owner := login(nil)
	request("GET", "/api/matches", nil, owner, http.StatusConflict)
	p1 := profileInput{Institution: institutions[0], Name: "Owner", Field: "061", From: "2097-09", To: "2098-01",
		Preferences: []preference{{Scope: "country", Value: "DE"}, {Scope: "region_group", Value: "DACH"}}}
	request("PUT", "/api/profile", p1, owner, http.StatusNoContent)
	bad := p1
	bad.Name, bad.Preferences = "Would overwrite", []preference{{Scope: "country", Value: "INVALID"}}
	request("PUT", "/api/profile", bad, owner, http.StatusBadRequest)
	if p := readProfile(owner); p["name"] != "Owner" || p["email"] != email || p["to"] != "2098-01" || len(p["preferences"].([]any)) != 2 {
		t.Fatalf("profile did not retain its saved values")
	}
	owner = login(owner)
	if readProfile(owner)["name"] != "Owner" {
		t.Fatal("rotated session lost the owner's profile")
	}

	other := login(nil)
	if readProfile(other)["name"] != "" {
		t.Fatal("a new session opened another account with the same unverified email")
	}
	p2 := profileInput{Institution: institutions[1], Name: "Other", Field: "071", From: "2097-12", To: "2098-06"}
	request("PUT", "/api/profile", p2, other, http.StatusNoContent)
	if readProfile(owner)["name"] != "Owner" {
		t.Fatal("saving another account changed the owner's profile")
	}
	if prefs, ok := readProfile(other)["preferences"].([]any); !ok || len(prefs) != 0 {
		t.Fatalf("new profile should have an empty destination list: %+v", prefs)
	}

	request("GET", "/api/catalog", nil, owner, http.StatusOK)
	search := request("GET", "/api/institutions?q="+url.QueryEscape(tag), nil, owner, http.StatusOK)
	if !strings.Contains(search.Body.String(), "Test campus") {
		t.Fatalf("university lookup did not return the test institution: %s", search.Body)
	}
	type matchPreview struct {
		ID, Name, OverlapFrom, OverlapTo string
		Sent, Received                   bool
		Preferences                      []preference
	}
	matches := func(cookie *http.Cookie, filter string) []matchPreview {
		t.Helper()
		w := request("GET", "/api/matches?"+filter, nil, cookie, http.StatusOK)
		if strings.Contains(w.Body.String(), email) {
			t.Fatal("match results exposed an email address")
		}
		var response struct{ Results []matchPreview }
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Results
	}
	if len(matches(owner, "")) != 0 || len(matches(owner, "field=all&minimum=3")) != 0 {
		t.Fatal("matching ignored the field or minimum overlap filter")
	}
	got := matches(owner, "field=all")
	if len(got) != 1 || got[0].Preferences == nil || len(got[0].Preferences) != 0 {
		t.Fatalf("worldwide search should show candidates with an empty destination list: %+v", got)
	}
	p2.Preferences = []preference{{Scope: "country", Value: "NL"}}
	request("PUT", "/api/profile", p2, other, http.StatusNoContent)
	got = matches(owner, "field=all&region=eu")
	if len(got) != 1 || got[0].Name != "Other" || got[0].OverlapFrom != "2097-12" || got[0].OverlapTo != "2098-01" ||
		len(got[0].Preferences) != 1 || got[0].Preferences[0] != p2.Preferences[0] {
		t.Fatalf("unexpected matching results: %+v", got)
	}
	if len(matches(owner, "field=all&region=dach")) != 0 {
		t.Fatal("region search used saved destinations rather than the candidate's university location")
	}
	request("PUT", "/api/interests/"+got[0].ID, nil, owner, http.StatusNoContent)
	incoming := matches(other, "field=all")
	if len(incoming) != 1 || !incoming[0].Received {
		t.Fatalf("recipient did not see the contact request: %+v", incoming)
	}
	request("PUT", "/api/interests/"+incoming[0].ID, nil, other, http.StatusNoContent)
	got = matches(owner, "field=all")
	if len(got) != 1 || !got[0].Sent || !got[0].Received {
		t.Fatalf("mutual interest was not recorded: %+v", got)
	}
	withoutDestinations := p1
	withoutDestinations.Preferences = []preference{}
	request("PUT", "/api/profile", withoutDestinations, owner, http.StatusNoContent)
	if prefs, ok := readProfile(owner)["preferences"].([]any); !ok || len(prefs) != 0 {
		t.Fatalf("clearing destinations should remove all saved preferences: %+v", prefs)
	}
	withoutDates := p1
	withoutDates.To = ""
	request("PUT", "/api/profile", withoutDates, owner, http.StatusBadRequest)
	if readProfile(owner)["to"] != p1.To {
		t.Fatal("a partial date range replaced the existing availability")
	}
	withoutDates.From = ""
	request("PUT", "/api/profile", withoutDates, owner, http.StatusNoContent)
	if p := readProfile(owner); p["from"] != "" || p["to"] != "" {
		t.Fatal("clearing both dates did not remove the saved availability")
	}
	p2.To = "2098-01"
	request("PUT", "/api/profile", p2, other, http.StatusNoContent)
	got = matches(owner, "field=all&minimum=6")
	if len(got) != 1 || got[0].OverlapFrom != p2.From || got[0].OverlapTo != p2.To {
		t.Fatalf("open availability did not match the candidate's full period: %+v", got)
	}
	if len(matches(owner, "")) != 0 || len(matches(owner, "field=all&region=dach")) != 0 ||
		len(matches(owner, "field=all&from=2096-01&to=2096-06")) != 0 {
		t.Fatal("open availability ignored a field, region, or explicitly selected period filter")
	}
	p2.From, p2.To = "", ""
	request("PUT", "/api/profile", p2, other, http.StatusNoContent)
	got = matches(owner, "field=all")
	if len(got) != 1 || got[0].OverlapFrom != "" || got[0].OverlapTo != "" {
		t.Fatalf("two open profiles should match without a fixed overlap: %+v", got)
	}
	request("PUT", "/api/profile", p1, owner, http.StatusNoContent)
	got = matches(owner, "field=all")
	if len(got) != 1 || got[0].OverlapFrom != p1.From || got[0].OverlapTo != p1.To {
		t.Fatalf("an open candidate did not match the viewer's fixed period: %+v", got)
	}
	if len(matches(owner, "field=all&minimum=6")) != 0 {
		t.Fatal("an open candidate met a minimum longer than the viewer's fixed period")
	}

	t.Run("independent search filters", func(t *testing.T) {
		third := login(nil)
		p3 := profileInput{Institution: institutions[2], Name: "North American candidate", Field: "021", From: "2098-07", To: "2098-12"}
		request("PUT", "/api/profile", p3, third, http.StatusNoContent)
		const fields = "field=071&field=021"
		const anyTime = "&from=&to="

		if got := matches(owner, fields+anyTime); len(got) != 2 {
			t.Fatalf("any-time search should include both selected fields beyond the saved dates: %+v", got)
		}
		if got := matches(owner, "field=071"+anyTime); len(got) != 1 || got[0].Name != p2.Name {
			t.Fatalf("removing a field should exclude that field's candidate: %+v", got)
		}
		if got := matches(owner, "field=all"+anyTime); len(got) != 2 {
			t.Fatalf("clearing all fields should allow all candidates: %+v", got)
		}
		for _, region := range []string{"AF", "AN", "AS", "NA", "OC", "SA", "europe"} {
			got := matches(owner, fields+anyTime+"&region="+region)
			switch region {
			case "NA":
				if len(got) != 1 || got[0].Name != p3.Name {
					t.Fatalf("North America search returned %+v", got)
				}
			case "europe":
				if len(got) != 1 || got[0].Name != p2.Name {
					t.Fatalf("Europe search returned %+v", got)
				}
			default:
				if len(got) != 0 {
					t.Fatalf("unexpected results in %s: %+v", region, got)
				}
			}
		}
		got := matches(owner, fields+"&region=NA&from=2098-09&to=2098-10")
		if len(got) != 1 || got[0].OverlapFrom != "2098-09" || got[0].OverlapTo != "2098-10" {
			t.Fatalf("custom period should override the saved availability in search: %+v", got)
		}
		if got := matches(owner, fields+"&region=NA&from=2098-01&to=2098-02"); len(got) != 0 {
			t.Fatalf("custom period should exclude candidates outside the selected dates: %+v", got)
		}
		request("GET", "/api/matches?field=071&field=bad", nil, owner, http.StatusBadRequest)
		request("GET", "/api/matches?from=2098-09&to=", nil, owner, http.StatusBadRequest)
		if saved := readProfile(owner); saved["field"] != p1.Field || saved["from"] != p1.From || saved["to"] != p1.To {
			t.Fatalf("changing search filters changed the saved profile: %+v", saved)
		}
	})

	request("DELETE", "/api/session", nil, owner, http.StatusNoContent)
	request("GET", "/api/profile", nil, owner, http.StatusUnauthorized)
}
