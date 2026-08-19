//go:build !windows

// Linux / macOS webview bootstrap for w2e's own GUI. Uses the
// cross-platform github.com/webview/webview_go binding, which dispatches to
// WebKit2GTK on Linux and WKWebView on macOS. Requires CGO + the platform
// development headers (WebKit2GTK-dev / Xcode CLT).
package webview

import (
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	webview "github.com/webview/webview_go"

	"github.com/minfu/w2e/internal/browser"
	w2eruntime "github.com/minfu/w2e/internal/runtime"
)

// Options describe a webview window launch.
type Options struct {
	URL       string
	Title     string
	Width     int
	Height    int
	Resizable bool
	AppID     string
	Debug     bool
	OnReady   func()
	OnShutdown func()
}

// App is the live webview handle (window).
type App struct {
	wv      webview.WebView
	opts    Options
	detDesc w2eruntime.Detection
}

// New constructs the webview window. Returns an error if the underlying
// webview runtime is missing.
func New(opts Options) (*App, error) {
	opts = withDefaults(opts)
	dataDir := defaultDataDir(opts.AppID)
	_ = os.MkdirAll(dataDir, 0o755)
	det := w2eruntime.Detect()

	w := webview.New(opts.Debug)
	if w == nil {
		return nil, fmt.Errorf("webview: unavailable (runtime %s). "+
			"Install WebKit2GTK (Linux) or use macOS 10.13+.", det.Kind)
	}
	app := &App{wv: w, opts: opts, detDesc: det}

	w.Init(`(function(){
		var _o = window.open;
		window.open = function(u) {
			if (u && u.indexOf('http://127.0.0.1') !== 0 && u.indexOf('http://localhost') !== 0) {
				return _o.apply(this, arguments);
			}
			return _o.apply(this, arguments);
		};
	})();`)
	_ = w.Bind("openExternal", func(url string) {
		if looksExternal(url, opts.URL) {
			_ = browser.Open(url)
		}
	})

	w.SetTitle(opts.Title)
	if opts.Resizable {
		w.SetSize(opts.Width, opts.Height, webview.HintNone)
	} else {
		w.SetSize(opts.Width, opts.Height, webview.HintFixed)
	}

	if opts.OnReady != nil {
		opts.OnReady()
	}
	w.Navigate(opts.URL)
	return app, nil
}

func (a *App) Run() {
	a.wv.Run()
	if a.opts.OnShutdown != nil {
		a.opts.OnShutdown()
	}
}

func (a *App) Eval(js string)                 { a.wv.Eval(js) }
func (a *App) Bind(name string, f interface{}) error { return a.wv.Bind(name, f) }
func (a *App) Terminate()                     { a.wv.Terminate() }
func (a *App) Destroy()                       { a.wv.Destroy() }
func (a *App) CloseWindow()                   { a.wv.Destroy() }
func (a *App) Detection() w2eruntime.Detection { return a.detDesc }

// WindowHandle returns 0 on non-Windows (native dialogs are stubbed out).
func (a *App) WindowHandle() uintptr { return 0 }

// Dispatch runs f on the main thread. On non-Windows, the underlying webview
// already runs on the main thread so we execute f directly.
func (a *App) Dispatch(f func()) { f() }

func withDefaults(o Options) Options {
	if o.Width <= 0 {
		o.Width = 1024
	}
	if o.Height <= 0 {
		o.Height = 720
	}
	if o.Width < 320 {
		o.Width = 320
	}
	if o.Height < 240 {
		o.Height = 240
	}
	if o.Title == "" {
		o.Title = "w2e"
	}
	if o.AppID == "" {
		o.AppID = deriveAppID()
	}
	return o
}

func defaultDataDir(appID string) string {
	switch goruntime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "w2e", appID)
	default: // linux / bsd
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "w2e", appID)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "w2e", appID)
	}
}

func deriveAppID() string {
	exe, err := os.Executable()
	if err != nil {
		return "app"
	}
	base := filepath.Base(exe)
	s := ""
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			s += string(r)
		}
	}
	if s == "" {
		return "app"
	}
	return s
}

func looksExternal(u string, localOrigin string) bool {
	if u == "" {
		return false
	}
	if contains(u, "://127.0.0.1") || contains(u, "://localhost") || contains(u, "://[::1]") {
		_ = localOrigin
		return false
	}
	if startsWithAny(u, "data:", "about:blank", "blob:") {
		return false
	}
	return true
}

func startsWithAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
