package main

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"

	"github.com/7thCode/morpho"
)

// maxBodyBytes caps request bodies. A corpus can be large, but not unbounded.
const maxBodyBytes = 8 << 20

type server struct {
	analyzer *morpho.Analyzer
	dictPath string
	origins  map[string]bool
}

func newServer(analyzer *morpho.Analyzer, dictPath string, origins []string) *server {
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}
	return &server{analyzer: analyzer, dictPath: dictPath, origins: allowed}
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.route(s.health, http.MethodGet))
	mux.HandleFunc("/analyze", s.route(s.analyze, http.MethodPost))
	mux.HandleFunc("/train", s.route(s.train, http.MethodPost))
	mux.HandleFunc("/stats", s.route(s.stats, http.MethodGet))
	mux.HandleFunc("/entries", s.route(s.entries, http.MethodGet))
	mux.HandleFunc("/word", s.route(s.word, http.MethodPost, http.MethodPut, http.MethodDelete))
	return mux
}

// route wraps a handler with CORS, method checking, and request-body limits.
//
// Cross-origin access is granted only to allow-listed origins; a request from
// any other browser origin is refused outright rather than merely left
// unreadable, since a hostile web page can still fire "simple" requests at a
// local server. Requests without an Origin header (curl, the desktop app)
// are unaffected. Requests with a body must be application/json, which makes
// browsers send a CORS preflight that the allow-list then rejects.
func (s *server) route(h http.HandlerFunc, methods ...string) http.HandlerFunc {
	allowedMethods := make(map[string]bool, len(methods))
	for _, m := range methods {
		allowedMethods[m] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if !s.origins[origin] {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if !allowedMethods[r.Method] {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// decode parses the JSON request body into v, answering 413/400 itself on failure.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		}
		return false
	}
	return true
}

// fail maps an analyzer error to a status: caller mistakes are 400, the rest 500.
func fail(w http.ResponseWriter, err error) {
	if errors.Is(err, morpho.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) analyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &req) {
		return
	}
	morphemes, err := s.analyzer.Analyze(req.Text)
	if err != nil {
		fail(w, err)
		return
	}
	if morphemes == nil {
		morphemes = []morpho.Morpheme{}
	}
	writeJSON(w, map[string]any{"morphemes": morphemes})
}

func (s *server) train(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Corpus string `json:"corpus"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := s.analyzer.Train(req.Corpus); err != nil {
		fail(w, err)
		return
	}
	if err := s.analyzer.Save(s.dictPath); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"word_count": s.analyzer.WordCount(),
		"is_trained": s.analyzer.IsTrained(),
		"pos_tags":   s.analyzer.POSTags(),
	})
}

func (s *server) entries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.analyzer.Entries())
}

func (s *server) word(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		surface := r.URL.Query().Get("surface")
		if surface == "" {
			writeError(w, http.StatusBadRequest, "surface parameter is required")
			return
		}
		if err := s.analyzer.DeleteWord(surface); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
		return
	}

	var req struct {
		Surface string `json:"surface"`
		POS     string `json:"pos"`
		Freq    int    `json:"freq"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := s.analyzer.SaveWord(req.Surface, req.POS, req.Freq); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}
