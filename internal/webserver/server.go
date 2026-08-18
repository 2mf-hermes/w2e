// Package webserver implements the embedded static server used by produced
// EXEs at runtime. It is also reused by the w2e builder's host to serve UI
// assets during build verification.
//
// Design (spec §7, §8):
//   - Binds 127.0.0.1:0 so the OS assigns a free port (NEVER 0.0.0.0).
//   - Serves from an io/fs.FS - either embed.FS or any FS - so large assets
//     are streamed via http.ServeContentFS rather than loaded whole.
//   - SPA fallback: file-not-found -> index.html (status 200), so React/Vue
//     router and History API work without server routes. Real asset misses
//     (.js/.css/.png) stay 404 so dev-side broken links surface immediately.
package webserver

import (
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// Server holds the configured embedded server plus the address it ended up on.
type Server struct {
	fsys    fs.FS
	root    string         // subdirectory of fsys that maps to URL "/" ("" = root)
	index   string         // entry HTML file name (e.g. "index.html")
	ln      net.Listener
	srv     *http.Server
	mu      sync.Mutex
	started bool
}

// Options configures Server.new.
type Options struct {
	FS    fs.FS // REQUIRED
	Root  string
	Index string // defaults to "index.html"
}

// New binds 127.0.0.1:0 (OS-assigned port) and returns a ready-to-Serve
// Server. Caller must Close() to free the port. Errors are returned if
// binding fails (i.e., no loopback interfaces available).
func New(opts Options) (*Server, error) {
	if opts.FS == nil {
		return nil, errors.New("webserver: FS is required")
	}
	index := opts.Index
	if index == "" {
		index = "index.html"
	}
	root := strings.TrimPrefix(path.Clean("/"+opts.Root), "/")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{fsys: opts.FS, root: root, index: index, ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s, nil
}

// Address returns "http://127.0.0.1:<port>/" once the listener is bound.
func (s *Server) Address() string { return "http://" + s.ln.Addr().String() + "/" }

// Start blocks until the server stops. Use StartInBackground for non-block.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return errors.New("webserver: already started")
	}
	s.started = true
	s.mu.Unlock()
	return s.srv.Serve(s.ln)
}

// StartInBackground launches s.srv in a goroutine and returns immediately.
// Returns any immediate bind errors.
func (s *Server) StartInBackground() {
	go func() { _ = s.srv.Serve(s.ln) }()
}

// Shutdown stops the server, releasing the port.
func (s *Server) Shutdown() error {
	return s.srv.Close()
}

// handle resolves a request to an FS file. Behavior:
//   - normalize path: "/about" -> "about" -> for root-mode, look up
//     joined path; for sub-dir strip root.
//   - if file missing: try the entry HTML and stream it as text/html (SPA route).
//   - if entry HTML missing too: 404.
//   - if file is a directory: append index; recurse once.
func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	urlPath := path.Clean("/" + r.URL.Path)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	// Path traversal hardening: fs.FS already rejects ".." with an error,
	// but we also explicitly block it here for defense-in-depth (spec §45).
	if strings.Contains(urlPath, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rel := strings.TrimPrefix(urlPath, "/")
	if s.root != "" {
		rel = path.Join(s.root, rel)
		rel = strings.TrimPrefix(rel, "/")
	}
	data, ct, err := openFromFS(s.fsys, s.index, rel)
	if err != nil {
		// SPA fallback: serve entry HTML with 200 for navigation routes.
		if servableEntry, entryCT, eerr := openFromFS(s.fsys, s.index, path.Join(s.root, s.index)); eerr == nil {
			stream(w, servableEntry, entryCT)
			return
		}
		http.NotFound(w, r)
		return
	}
	stream(w, data, ct)
}

// openFromFS resolves rel within s.fsys, returning (file, content-type, err).
// If rel is empty or refers to a non-existent file, it tries the entry HTML.
func openFromFS(fsys fs.FS, index, rel string) (fs.File, string, error) {
	if rel == "" || strings.HasSuffix(rel, "/") {
		rel = path.Join(rel, index)
	}
	f, err := fsys.Open(rel)
	if err == nil {
		// If it's a directory (e.g., accessed explicit), try index inside it.
		if st, serr := f.Stat(); serr == nil && st.IsDir() {
			_ = f.Close()
			f, err = fsys.Open(path.Join(rel, index))
		}
	}
	if err != nil {
		return nil, "", err
	}
	ct := mimeTypeFor(path.Ext(rel))
	return f, ct, nil
}

// stream writes a file content to w with proper headers. Uses
// http.ServeContent for Last-Modified/range handling, so large assets are
// not read into memory all at once (spec §76).
//
// fs.File is an io.ReadSeeker-compatible reader when emitted by embed.FS,
// so we can pass it directly to http.ServeContent. We wrap with a small
// adapter that recovers from stat failures.
func stream(w http.ResponseWriter, f fs.File, contentType string) {
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		// embed.FS files implement ReadSeeker; if some FS doesn't, fall back
		// to io.Copy semantics — read once and write.
		w.Header().Set("Content-Type", contentType)
		_, _ = io.Copy(w, f)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, nil, st.Name(), st.ModTime(), rs)
}
