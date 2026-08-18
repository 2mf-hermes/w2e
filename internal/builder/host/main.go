// Package main is the generic host runtime embedded in w2e.exe.
// It reads its configuration from w2e-config.json and serves web content
// from the web/ subdirectory — no compilation required at packaging time.
// This allows w2e to produce self-contained Windows EXEs without the
// end-user having Go installed.
//
// Build:  go build -o ../host.exe .
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
)

// Config is the JSON schema written by w2e next to the host EXE.
type Config struct {
	AppID     string `json:"app_id"`
	Title     string `json:"title"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Resizable bool   `json:"resizable"`
	EntryFile string `json:"entry_file"`
	Debug     bool   `json:"debug"`
}

// Trailer format: [4-byte uint32 LE zip-length] [8-byte magic]
// Appended after the PE file to make a self-extracting EXE.
var trailerMagic = []byte("W2E ZIP1")

func defaultConfig() Config {
	return Config{
		AppID:     "w2eapp",
		Title:     "w2e App",
		Width:     1024,
		Height:    768,
		Resizable: true,
		EntryFile: "index.html",
	}
}

func main() {
	runtime.LockOSThread()

	exePath, err := os.Executable()
	if err != nil {
		showError("Cannot determine executable path: " + err.Error())
		os.Exit(1)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	exeDir := filepath.Dir(exePath)

	// Determine data directory: either from appended zip or from local web/ dir.
	dataDir, cleanup := resolveDataDir(exePath, exeDir)
	if cleanup != nil {
		defer cleanup()
	}

	// Read config.
	cfg := defaultConfig()
	cfgPath := filepath.Join(dataDir, "w2e-config.json")
	if raw, err := os.ReadFile(cfgPath); err == nil {
		_ = json.Unmarshal(raw, &cfg)
	}
	if cfg.AppID == "" {
		cfg.AppID = "w2eapp"
	}
	if cfg.EntryFile == "" {
		cfg.EntryFile = "index.html"
	}
	if cfg.Width <= 0 {
		cfg.Width = 1024
	}
	if cfg.Height <= 0 {
		cfg.Height = 768
	}

	// Serve web content.
	webDir := filepath.Join(dataDir, "web")
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		showError("Web content directory not found: " + webDir)
		os.Exit(2)
	}

	srv, err := newFileServer(webDir, cfg.EntryFile)
	if err != nil {
		showError("Failed to start local server: " + err.Error())
		os.Exit(3)
	}
	go srv.ListenAndServe()
	defer srv.Close()

	waitForServer(srv.Addr(), 5*time.Second)

	// Open WebView2 window.
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:    cfg.Debug,
		DataPath: userDataFolder(cfg.AppID),
		WindowOptions: webview2.WindowOptions{
			Title:  cfg.Title,
			Width:  uint(cfg.Width),
			Height: uint(cfg.Height),
			Center: true,
		},
	})
	if w == nil {
		srv.Close()
		showError("WebView2 Runtime not found.\nInstall Microsoft Edge WebView2 Runtime from:\nhttps://developer.microsoft.com/en-us/microsoft-edge/webview2/")
		os.Exit(4)
	}
	if !cfg.Resizable {
		w.SetSize(cfg.Width, cfg.Height, webview2.HintFixed)
	} else {
		w.SetSize(960, 600, webview2.HintMin)
	}
	installNavGuard(w, srv.Addr())
	w.Navigate("http://" + srv.Addr() + "/")
	w.Run()
}

// ---------------------------------------------------------------------------
// Data directory resolution (appended zip or local)
// ---------------------------------------------------------------------------

// resolveDataDir returns the directory containing w2e-config.json and web/.
// If the EXE has appended zip data, it extracts to a temp cache first.
func resolveDataDir(exePath, exeDir string) (string, func()) {
	// Try appended data first.
	if zipData := readAppendedZip(exePath); zipData != nil {
		cacheDir := cacheDirFor(exePath)
		hashFile := filepath.Join(cacheDir, ".w2e_hash")
		currentHash := fmt.Sprintf("%x", sha256.Sum256(zipData))
		if raw, err := os.ReadFile(hashFile); err == nil && string(raw) == currentHash {
			return cacheDir, nil
		}
		// Extract.
		os.MkdirAll(cacheDir, 0o755)
		if err := extractZipTo(zipData, cacheDir); err == nil {
			_ = os.WriteFile(hashFile, []byte(currentHash), 0o644)
			return cacheDir, func() {
				// Don't remove cache — it's reused.
			}
		}
	}

	// Fallback: read from same directory as EXE.
	return exeDir, nil
}

// readAppendedZip reads a zip archive appended to the end of the EXE file.
// Returns nil if no appended data is found.
func readAppendedZip(exePath string) []byte {
	f, err := os.Open(exePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil
	}
	size := stat.Size()

	trailerLen := int64(4 + len(trailerMagic)) // 4 bytes length + magic
	if size < trailerLen {
		return nil
	}

	// Read the trailer.
	trailer := make([]byte, trailerLen)
	if _, err := f.ReadAt(trailer, size-trailerLen); err != nil {
		return nil
	}
	if !bytes.Equal(trailer[4:], trailerMagic) {
		return nil
	}

	dataLen := binary.LittleEndian.Uint32(trailer[:4])
	if dataLen == 0 || int64(dataLen)+trailerLen > size {
		return nil
	}

	dataStart := size - trailerLen - int64(dataLen)
	zipData := make([]byte, dataLen)
	if _, err := f.ReadAt(zipData, dataStart); err != nil {
		return nil
	}
	return zipData
}

func extractZipTo(data []byte, destDir string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		// Prevent zip-slip.
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0o755)
		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func cacheDirFor(exePath string) string {
	h := sha256.Sum256([]byte(exePath))
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = filepath.Join(os.TempDir(), "w2e")
	}
	return filepath.Join(la, "w2e", "cache", fmt.Sprintf("%x", h[:8]))
}

// ---------------------------------------------------------------------------
// HTTP file server
// ---------------------------------------------------------------------------

type fileServer struct {
	ln    net.Listener
	srv   *http.Server
	root  string
	entry string
}

func newFileServer(root, entry string) (*fileServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	fs := &fileServer{ln: ln, root: root, entry: entry}
	mux := http.NewServeMux()
	mux.HandleFunc("/", fs.handle)
	fs.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return fs, nil
}

func (s *fileServer) Addr() string         { return s.ln.Addr().String() }
func (s *fileServer) ListenAndServe() error { return s.srv.Serve(s.ln) }
func (s *fileServer) Close() error          { return s.srv.Close() }

func (s *fileServer) handle(w http.ResponseWriter, r *http.Request) {
	urlPath := path.Clean("/" + r.URL.Path)
	if strings.Contains(urlPath, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	rel := strings.TrimPrefix(urlPath, "/")
	if rel == "" || strings.HasSuffix(rel, "/") {
		rel = path.Join(rel, s.entry)
	}

	fpath := filepath.Join(s.root, filepath.FromSlash(rel))
	// If it's a directory, serve the entry file.
	if info, err := os.Stat(fpath); err == nil && info.IsDir() {
		fpath = filepath.Join(fpath, s.entry)
	}

	f, err := os.Open(fpath)
	if err != nil {
		// SPA fallback: serve the entry file for any path.
		fallback := filepath.Join(s.root, s.entry)
		if fb, ferr := os.Open(fallback); ferr == nil {
			defer fb.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, s.entry, time.Time{}, fb)
			return
		}
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	ct := mimeFor(filepath.Ext(fpath))
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-cache")
	info, _ := f.Stat()
	http.ServeContent(w, r, filepath.Base(fpath), info.ModTime(), f)
}

func mimeFor(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "", "html", "htm":
		return "text/html; charset=utf-8"
	case "css":
		return "text/css; charset=utf-8"
	case "js", "mjs":
		return "application/javascript; charset=utf-8"
	case "json":
		return "application/json; charset=utf-8"
	case "svg":
		return "image/svg+xml"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	case "woff":
		return "font/woff"
	case "woff2":
		return "font/woff2"
	case "ttf":
		return "font/ttf"
	case "otf":
		return "font/otf"
	case "eot":
		return "application/vnd.ms-fontobject"
	case "wasm":
		return "application/wasm"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mp3":
		return "audio/mpeg"
	case "pdf":
		return "application/pdf"
	case "txt":
		return "text/plain; charset=utf-8"
	case "xml":
		return "application/xml; charset=utf-8"
	}
	return "application/octet-stream"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func waitForServer(addr string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func showError(msg string) {
	procMessageBox := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("w2e")
	text, _ := syscall.UTF16PtrFromString(msg)
	procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0)
}

func installNavGuard(w webview2.WebView, localAddr string) {
	policy := `window.__w2eLocalAddr = "` + localAddr + `";(function(){
var inLocal = function(u){
  if(!u) return true;
  if(u.indexOf('http://127.0.0.1') === 0 || u.indexOf('http://localhost') === 0) return true;
  return (u.indexOf('data:') === 0) || (u.indexOf('about:blank') === 0) || (u.indexOf('blob:') === 0) || (u.indexOf('javascript:') === 0);
};
var open = window.open;
window.open = function(url){
  try { if(!inLocal(url)) { window.__ext(url); return null; } } catch(e) {}
  return open && open.apply(this, arguments);
};
document.addEventListener('click', function(ev){
  var a = ev.target && ev.target.closest ? ev.target.closest('a') : null;
  if(!a) return;
  var href = a.getAttribute('href');
  if(!href) return;
  if(!inLocal(href) && (a.target === '_blank' || (ev.metaKey || ev.ctrlKey))) {
    ev.preventDefault();
    try { window.__ext(href); } catch(e) {}
  }
}, true);
})();`
	w.Init(policy)
	_ = w.Bind("__ext", func(url string) {
		openBrowser(url)
	})
}

func openBrowser(u string) {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return
	}
	_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u).Start()
}

func userDataFolder(appID string) string {
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = filepath.Join(os.TempDir(), "w2e")
	}
	return filepath.Join(la, "w2e", appID)
}

// Ensure binary.LittleEndian and bytes are used for the trailer format.
var _ = binary.LittleEndian
