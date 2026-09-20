package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/7thCode/morpho"
)

// defaultOrigins are the browser origins allowed to call the API: the Vite
// dev server, and "null", which is what a page loaded from file:// (the
// packaged Electron app) sends.
const defaultOrigins = "http://localhost:5173,http://127.0.0.1:5173,null"

func main() {
	host := flag.String("host", "127.0.0.1", "address to listen on (use 0.0.0.0 to expose the API to the network)")
	port := flag.Int("port", 8765, "HTTP port")
	dictPath := flag.String("dict", "dict.json", "path to dictionary JSON file")
	origins := flag.String("cors-origins", defaultOrigins, "comma-separated browser origins allowed to call the API")
	flag.Parse()

	analyzer, backup, err := morpho.OpenOrRecover(*dictPath)
	if err != nil {
		log.Fatal(err)
	}
	if backup != "" {
		log.Printf("dictionary %s was corrupt; moved to %s and started empty", *dictPath, backup)
	}

	s := newServer(analyzer, *dictPath, splitList(*origins))

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute, // training large corpora takes a while
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("morpho server listening on %s (dict: %s)", addr, *dictPath)
	log.Fatal(srv.ListenAndServe())
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
