package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/7thCode/morpho"
)

type server struct {
	analyzer *morpho.Analyzer
	dictPath string
}

func (s *server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *server) analyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	morphemes, err := s.analyzer.Analyze(req.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"morphemes": morphemes})
}

func (s *server) train(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Corpus string `json:"corpus"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.analyzer.Train(req.Corpus); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.analyzer.Save(s.dictPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func main() {
	port := flag.Int("port", 8765, "HTTP port")
	dictPath := flag.String("dict", "dict.json", "path to dictionary JSON file")
	flag.Parse()

	analyzer, err := morpho.New(*dictPath)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{analyzer: analyzer, dictPath: *dictPath}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.cors(s.health))
	mux.HandleFunc("/analyze", s.cors(s.analyze))
	mux.HandleFunc("/train", s.cors(s.train))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("morpho server listening on %s (dict: %s)", addr, *dictPath)
	log.Fatal(http.ListenAndServe(addr, mux))
}
