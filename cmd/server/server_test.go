package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/7thCode/morpho"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dict.json")
	a, err := morpho.New(path)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(newServer(a, path, splitList(defaultOrigins)).handler())
	t.Cleanup(ts.Close)
	return ts, path
}

func do(t *testing.T, method, url, body string, headers map[string]string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, sb.String()
}

var jsonHeader = map[string]string{"Content-Type": "application/json"}

func TestAnalyzeTrainAndWordRoundTrip(t *testing.T) {
	ts, path := newTestServer(t)

	resp, body := do(t, "POST", ts.URL+"/train", `{"corpus":"東京は日本の首都です。"}`, jsonHeader)
	if resp.StatusCode != 200 {
		t.Fatalf("train: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"ok":true`) {
		t.Errorf("train body = %s", body)
	}

	resp, body = do(t, "POST", ts.URL+"/word", `{"surface":"東京タワー","pos":"名詞","freq":1}`, jsonHeader)
	if resp.StatusCode != 200 {
		t.Fatalf("word: %d %s", resp.StatusCode, body)
	}

	resp, body = do(t, "POST", ts.URL+"/analyze", `{"text":"東京タワー"}`, jsonHeader)
	var out struct {
		Morphemes []morpho.Morpheme `json:"morphemes"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || resp.StatusCode != 200 {
		t.Fatalf("analyze: %d %s (%v)", resp.StatusCode, body, err)
	}
	if len(out.Morphemes) != 1 || out.Morphemes[0].Surface != "東京タワー" {
		t.Errorf("morphemes = %+v", out.Morphemes)
	}

	// Empty text yields an empty list, not null.
	_, body = do(t, "POST", ts.URL+"/analyze", `{"text":""}`, jsonHeader)
	if !strings.Contains(body, `"morphemes":[]`) {
		t.Errorf("empty analyze body = %s", body)
	}

	resp, body = do(t, "DELETE", ts.URL+"/word?surface="+"東京タワー", "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("delete: %d %s", resp.StatusCode, body)
	}

	// Train persisted to disk.
	if a, err := morpho.New(path); err != nil || !a.IsTrained() {
		t.Errorf("dictionary not persisted after /train: %v", err)
	}
}

func TestInputErrorsAre4xx(t *testing.T) {
	ts, _ := newTestServer(t)
	tests := []struct {
		name, method, path, body string
		headers                  map[string]string
		want                     int
	}{
		{"bad JSON", "POST", "/analyze", `{`, jsonHeader, 400},
		{"unknown POS", "POST", "/word", `{"surface":"猫","pos":"noun","freq":1}`, jsonHeader, 400},
		{"whitespace word", "POST", "/word", `{"surface":"a b","pos":"名詞","freq":1}`, jsonHeader, 400},
		{"empty corpus", "POST", "/train", `{"corpus":""}`, jsonHeader, 400},
		{"delete without surface", "DELETE", "/word", "", nil, 400},
		{"wrong method", "GET", "/analyze", "", nil, 405},
		{"wrong method 2", "POST", "/stats", "{}", jsonHeader, 405},
		{"text/plain body", "POST", "/analyze", `{"text":"x"}`, map[string]string{"Content-Type": "text/plain"}, 415},
		{"no content type", "POST", "/analyze", `{"text":"x"}`, nil, 415},
	}
	for _, tt := range tests {
		resp, body := do(t, tt.method, ts.URL+tt.path, tt.body, tt.headers)
		if resp.StatusCode != tt.want {
			t.Errorf("%s: status %d (%s), want %d", tt.name, resp.StatusCode, body, tt.want)
		}
		if tt.want >= 400 && !strings.Contains(body, `"error"`) {
			t.Errorf("%s: error body is not JSON with an error field: %s", tt.name, body)
		}
	}
}

func TestBodyTooLarge(t *testing.T) {
	ts, _ := newTestServer(t)
	huge := `{"text":"` + strings.Repeat("あ", maxBodyBytes/3+10) + `"}`
	resp, _ := do(t, "POST", ts.URL+"/analyze", huge, jsonHeader)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", resp.StatusCode)
	}
}

func TestCORS(t *testing.T) {
	ts, _ := newTestServer(t)

	// Allowed origins are echoed back, including file:// pages ("null").
	for _, origin := range []string{"http://localhost:5173", "null"} {
		resp, _ := do(t, "GET", ts.URL+"/health", "", map[string]string{"Origin": origin})
		if resp.StatusCode != 200 || resp.Header.Get("Access-Control-Allow-Origin") != origin {
			t.Errorf("origin %q: status %d, ACAO %q", origin, resp.StatusCode, resp.Header.Get("Access-Control-Allow-Origin"))
		}
	}

	// Other web pages are refused outright, even for requests that need no preflight.
	resp, _ := do(t, "POST", ts.URL+"/analyze", `{"text":"x"}`, map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign origin: status %d, want 403", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("foreign origin was granted CORS access")
	}

	// Preflight.
	resp, _ = do(t, "OPTIONS", ts.URL+"/word", "", map[string]string{"Origin": "http://localhost:5173", "Access-Control-Request-Method": "DELETE"})
	if resp.StatusCode != http.StatusNoContent || !strings.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "DELETE") {
		t.Errorf("preflight: %d %v", resp.StatusCode, resp.Header)
	}
	resp, _ = do(t, "OPTIONS", ts.URL+"/word", "", map[string]string{"Origin": "https://evil.example"})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("foreign preflight: %d, want 403", resp.StatusCode)
	}

	// No Origin (curl, desktop app): allowed, no CORS headers.
	resp, _ = do(t, "GET", ts.URL+"/health", "", nil)
	if resp.StatusCode != 200 || resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("no-origin request: %d %v", resp.StatusCode, resp.Header)
	}
}

func TestStatsAndEntries(t *testing.T) {
	ts, _ := newTestServer(t)
	do(t, "POST", ts.URL+"/train", `{"corpus":"猫が鳴く。"}`, jsonHeader)

	_, body := do(t, "GET", ts.URL+"/stats", "", nil)
	var st struct {
		WordCount int      `json:"word_count"`
		IsTrained bool     `json:"is_trained"`
		POSTags   []string `json:"pos_tags"`
	}
	if err := json.Unmarshal([]byte(body), &st); err != nil || !st.IsTrained || st.WordCount == 0 || len(st.POSTags) == 0 {
		t.Errorf("stats = %s (%v)", body, err)
	}

	_, body = do(t, "GET", ts.URL+"/entries", "", nil)
	var entries []morpho.DictEntry
	if err := json.Unmarshal([]byte(body), &entries); err != nil || len(entries) == 0 {
		t.Errorf("entries = %s (%v)", body, err)
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" a, b ,,c ")
	if strings.Join(got, "|") != "a|b|c" {
		t.Errorf("splitList = %q", got)
	}
}
