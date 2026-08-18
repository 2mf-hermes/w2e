// Package ui hosts the embedded w2e desktop GUI: an Apple/iOS 26 Liquid Glass
// web UI presented in a WebView2 window.
//
// The UI is served from a localhost HTTP server (spec §12). All assets are
// embedded — no external CDN (spec §X): the user can launch the packager
// offline. The server listens on 127.0.0.1:0 (§7, §48) — never on 0.0.0.0.
//
// Build orchestration (the "開始打包" button) flows through the same
// builder.Engine that CLI/MCP use (§6).
package ui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/minfu/w2e/internal/builder"
	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/i18n"
	mainwebview "github.com/minfu/w2e/internal/webview"
	"github.com/minfu/w2e/internal/validator"
)

//go:embed assets/*
var uiAssets embed.FS

// noCache wraps an http.Handler to prevent browser/WebView2 from caching
// static assets. This ensures the latest JS/CSS is always loaded.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// Run launches the w2e desktop GUI. Blocks until the user closes the window.
func Run(ctx context.Context) error {
	log.SetFlags(0)
	log.SetPrefix("[w2e-ui] ")

	log.Println("initializing assets...")
	sub, err := fs.Sub(uiAssets, "assets")
	if err != nil {
		log.Printf("ERROR fs.Sub: %v", err)
		return err
	}
	log.Println("starting HTTP server on 127.0.0.1:0")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("ERROR net.Listen: %v", err)
		return err
	}
	defer ln.Close()
	log.Printf("HTTP server listening on %s", ln.Addr())

	state := &guiState{engine: builder.New(), bundle: i18n.Default()}
	mux := http.NewServeMux()
	mux.Handle("/", noCache(http.FileServerFS(sub)))
	mux.HandleFunc("/api/state", state.handleState)
	mux.HandleFunc("/api/validate", state.handleValidate)
	mux.HandleFunc("/api/build", state.handleBuild)
	mux.HandleFunc("/api/doctor", state.handleDoctor)
	mux.HandleFunc("/api/i18n", state.handleI18N)
	mux.HandleFunc("/api/log", state.handleLog)
	mux.HandleFunc("/api/pickDirectory", state.handlePickDirectory)
	mux.HandleFunc("/api/pickFile", state.handlePickFile)

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	// Detect screen size for a default window that fits.
	scrW, scrH := screenMetrics()
	defW, defH := 1180, 760
	if scrW >= 1920 && scrH >= 1080 {
		defW, defH = 1280, 800
	}
	if scrW >= 2560 && scrH >= 1440 {
		defW, defH = 1440, 900
	}
	log.Printf("creating WebView2 window (%dx%d)...", defW, defH)
	app, err := mainwebview.New(mainwebview.Options{
		Title:     "w2e",
		Width:     defW,
		Height:    defH,
		Resizable: true,
		AppID:     "w2e-gui",
		Debug:     true,
		URL:       "http://" + ln.Addr().String() + "/",
	})
	if err != nil {
		log.Printf("ERROR creating webview: %v", err)
		return fmt.Errorf("could not create webview window: %w", err)
	}
	state.app = app
	bindAPI(app, state)
	log.Println("WebView2 window created, entering Run()...")
	app.Run()
	log.Println("WebView2 Run() returned (window closed)")
	<-ctx.Done()
	return nil
}

func userDataFolder() string {
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = os.TempDir()
	}
	return la + string(os.PathSeparator) + "w2e" + string(os.PathSeparator) + "w2e-gui"
}

// guiState is the per-process UI state shared between the UI and the API
// handlers.
type guiState struct {
	engine *builder.Engine
	bundle *i18n.Bundle
	app    *mainwebview.App // set after the webview is created
}

// pickDir dispatches a folder picker to the WebView2 main thread so the
// native dialog gets the correct message pump.
func (s *guiState) pickDir() string {
	ch := make(chan string, 1)
	s.app.Dispatch(func() {
		ch <- pickDirectory(s.app.WindowHandle())
	})
	return <-ch
}

// pickIcon dispatches a file picker to the WebView2 main thread.
func (s *guiState) pickIcon() string {
	ch := make(chan string, 1)
	s.app.Dispatch(func() {
		ch <- pickFile(s.app.WindowHandle())
	})
	return <-ch
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *guiState) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Detect screen size and pick a default window size that fits the wizard.
	scrW, scrH := screenMetrics()
	defW, defH := 1024, 720
	if scrW > 0 && scrH > 0 {
		if scrH < 768 {
			defW = int(float64(scrW) * 0.80)
			defH = int(float64(scrH) * 0.80)
		} else {
			defW = int(float64(scrW) * 0.65)
			defH = int(float64(scrH) * 0.72)
			if defW > 1100 { defW = 1100 }
			if defH > 820 { defH = 820 }
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app_title":        "w2e",
		"current_locale":   s.bundle.Locale(),
		"available_locales": s.bundle.Available(),
		"default_width":    defW,
		"default_height":   defH,
		"min_width":        config.MinWidth,
		"min_height":       config.MinHeight,
		"screen_width":     scrW,
		"screen_height":    scrH,
		"version":          "1.0.0-dev",
	})
}

func (s *guiState) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SourceDir string `json:"source_dir"`
		Entry     string `json:"entry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	cfg := &config.BuildConfig{SourceDir: req.SourceDir, EntryFile: req.Entry}
	rep, err := validator.Validate(cfg)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *guiState) handleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg builder.BuildForm
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	bc := cfg.ToBuildConfig()
	res := s.engine.Build(bc, nil)
	// Always return the full MultiResult so the GUI's target-segmented summary
	// can render per-target success/failure side-by-side (e.g. Windows ✓ but
	// Linux ✕ due to a missing cross-C-compiler). The frontend Live region
	// surfaces the headline status; the report list shows per-target details.
	writeJSON(w, http.StatusOK, res)
}

func (s *guiState) handleDoctor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, builder.Doctor())
}

func (s *guiState) handleI18N(w http.ResponseWriter, r *http.Request) {
	// Switch locale if ?locale= provided
	if q := r.URL.Query().Get("locale"); q != "" {
		_ = s.bundle.SetLocale(q)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	all := s.bundle.All()
	out := map[string]string{}
	for _, k := range s.bundle.Keys() {
		out[k] = all[k]
	}
	_ = json.NewEncoder(w).Encode(out)
}

// bindAPI exposes Go functions to JavaScript. Bind-based pickers run on
// the WebView2 main thread (required for Win32 native dialogs that need a
// message pump). The HTTP /api/pickDirectory and /api/pickFile endpoints
// are kept as a fallback for environments where Bind is unavailable.
func bindAPI(app *mainwebview.App, s *guiState) {
	_ = app.Bind("__pickDirectory", func() string {
		return pickDirectory(app.WindowHandle())
	})
	_ = app.Bind("__pickFile", func() string {
		return pickFile(app.WindowHandle())
	})
}

// numeric helpers reused by the UI
func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

// handleLog receives console output and uncaught errors the page forwards
// while running inside the WebView2 host. We mirror them to the process
// stderr (which the GUI app forwards to the parent shell via AttachConsole)
// and append a line to %LOCALAPPDATA%\w2e\w2e-gui\ui.log so the user can
// grab it for diagnosis. Always returns 200.
func (s *guiState) handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Level  string `json:"level"`
		Source string `json:"source"`
		Line   int    `json:"line"`
		Col    int    `json:"col"`
		Msg    string `json:"msg"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "err": err.Error()})
		return
	}
	fmt.Fprintf(os.Stderr, "[ui:%s] %s (%s:%d:%d)\n", req.Level, req.Msg, req.Source, req.Line, req.Col)
	if f, err := os.OpenFile(userDataFolder()+string(os.PathSeparator)+"ui.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		fmt.Fprintf(f, "[%s] %s (%s:%d:%d)%s\n", req.Level, req.Msg, req.Source, req.Line, req.Col, "")
		_ = f.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handlePickDirectory opens a Windows folder picker and returns the chosen
// path; on non-Windows or failure returns an empty string so the JS can fall
// back to manual entry. Implemented in ui_dialog_windows.go / ui_dialog_stub.go.
func (s *guiState) handlePickDirectory(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeJSON(w, http.StatusOK, map[string]any{"path": s.pickDir()})
}

// handlePickFile opens a Windows file picker (for an icon) and returns the
// chosen path; on non-Windows or failure returns an empty string.
func (s *guiState) handlePickFile(w http.ResponseWriter, r *http.Request) {
	_ = r
	writeJSON(w, http.StatusOK, map[string]any{"path": s.pickIcon()})
}
var _ = atoi
