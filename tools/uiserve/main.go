// Quick static server for the w2e UI assets, so we can view the UI in a
// browser for visual verification. Serves GET /app.css, /app.js, /index.html
// and /api/state, /api/i18n with canned responses so the JS keeps working.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("dir", "internal/ui/assets", "assets dir")
	addr := flag.String("addr", "127.0.0.1:8765", "listen addr")
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		full := filepath.Join(*dir, path)
		data, err := os.ReadFile(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(path) {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"app_title":         "w2e",
			"current_locale":    "en",
			"available_locales": []string{"en", "zh-TW", "zh-CN", "ja", "ko"},
			"default_width":     1024,
			"default_height":    720,
			"min_width":         320,
			"min_height":        240,
			"version":           "1.0.0-dev",
		})
	})
	mux.HandleFunc("/api/i18n", func(w http.ResponseWriter, r *http.Request) {
		loc := r.URL.Query().Get("locale")
		if loc == "" {
			loc = "en"
		}
		data, err := os.ReadFile(filepath.Join("internal/i18n/locales", loc+".json"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
	})
	fmt.Printf("serving %s on http://%s/\n", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
