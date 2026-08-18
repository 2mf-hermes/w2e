//go:build windows

// Windows-specific webview bootstrap for w2e's own GUI. Uses the pure-Go
// github.com/jchv/go-webview2 binding so w2e.exe builds with CGO_ENABLED=0
// on Windows.
package webview

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"

	"github.com/minfu/w2e/internal/browser"
	"github.com/minfu/w2e/internal/runtime"
)

//go:embed w2e.ico
var appIconData []byte

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procSetIcon       = user32.NewProc("SendMessageW")
	procCreateIcon    = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon   = user32.NewProc("DestroyIcon")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	WM_SETICON        uintptr = 0x0080
	ICON_SMALL        uintptr = 0
	ICON_BIG          uintptr = 1
	SM_CXSCREEN       uintptr = 0
	SM_CYSCREEN       uintptr = 1
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
	wv      webview2.WebView
	opts    Options
	detDesc runtime.Detection
}

// New constructs the WebView2 window. Returns an error if WebView2 cannot
// be initialized.
func New(opts Options) (*App, error) {
	opts = withDefaults(opts)
	dataDir := defaultDataDir(opts.AppID)
	_ = os.MkdirAll(dataDir, 0o755)
	det := runtime.Detect()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:    opts.Debug,
		DataPath: dataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  opts.Title,
			Width:  uint(opts.Width),
			Height: uint(opts.Height),
			Center: true,
		},
	})
	if w == nil {
		return nil, fmt.Errorf("webview: WebView2 unavailable (runtime %s). "+
			"Install Microsoft WebView2 Runtime (Evergreen) to use w2e apps.",
			det.Kind)
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

	if !opts.Resizable {
		w.SetSize(opts.Width, opts.Height, webview2.HintFixed)
	} else {
		// Set minimum window size so the wizard card stays readable.
		w.SetSize(540, 500, webview2.HintMin)
	}

	// Set the window icon from the embedded .ico (taskbar + title bar).
	hwnd := uintptr(w.Window())
	setAppIcon(hwnd)

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

func (a *App) Eval(js string)                            { a.wv.Eval(js) }
func (a *App) Bind(name string, f interface{}) error      { return a.wv.Bind(name, f) }
func (a *App) Terminate()                                { a.wv.Terminate() }
func (a *App) Destroy()                                  { a.wv.Destroy() }
func (a *App) CloseWindow()                              { a.wv.Destroy() }
func (a *App) Detection() runtime.Detection               { return a.detDesc }

// WindowHandle returns the native Win32 HWND as a uintptr so callers can
// pass it as a parent/owner to native dialogs (file picker, folder picker).
func (a *App) WindowHandle() uintptr { return uintptr(a.wv.Window()) }

// Dispatch schedules f to run on the WebView2 main thread (message loop).
// Used to run Win32 dialogs that need the thread's message pump.
func (a *App) Dispatch(f func()) { a.wv.Dispatch(f) }

// getScreenSize returns the primary monitor resolution.
func getScreenSize() (w, h int) {
	ret, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	w = int(ret)
	ret, _, _ = procGetSystemMetrics.Call(SM_CYSCREEN)
	h = int(ret)
	return
}

func withDefaults(o Options) Options {
	screenW, screenH := getScreenSize()

	// Responsive defaults: fit the wizard card comfortably.
	// On small screens (< 768px height), use 80% of screen.
	// On large screens, cap at reasonable max so window isn't huge.
	defW := 1024
	defH := 720
	if screenW > 0 && screenH > 0 {
		if screenH < 768 {
			defW = int(float64(screenW) * 0.80)
			defH = int(float64(screenH) * 0.80)
		} else {
			// 75% of screen, capped at reasonable max
			defW = int(float64(screenW) * 0.65)
			defH = int(float64(screenH) * 0.72)
			if defW > 1100 {
				defW = 1100
			}
			if defH > 820 {
				defH = 820
			}
		}
	}

	if o.Width <= 0 {
		o.Width = defW
	}
	if o.Height <= 0 {
		o.Height = defH
	}
	if o.Width < 540 {
		o.Width = 540
	}
	if o.Height < 500 {
		o.Height = 500
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
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = filepath.Join(os.TempDir(), "w2e")
	}
	return filepath.Join(la, "w2e", appID)
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

// setAppIcon parses the embedded .ico and sets both ICON_SMALL and ICON_BIG
// on the window so the taskbar and title bar display the hexagonal w2e logo.
func setAppIcon(hwnd uintptr) {
	if len(appIconData) < 6 {
		return
	}
	// Parse ICONDIR
	_ = binary.LittleEndian.Uint16(appIconData[0:2]) // reserved
	imgType := binary.LittleEndian.Uint16(appIconData[2:4])
	count := binary.LittleEndian.Uint16(appIconData[4:6])
	if imgType != 1 || count == 0 {
		return
	}
	// Find the largest image for ICON_BIG and smallest for ICON_SMALL.
	type iconEntry struct {
		w, h   int
		data   []byte
		offset uint32
		size   uint32
	}
	entries := make([]iconEntry, count)
	for i := 0; i < int(count); i++ {
		off := 6 + i*16
		if off+16 > len(appIconData) {
			return
		}
		w := int(appIconData[off])
		if w == 0 {
			w = 256
		}
		h := int(appIconData[off+1])
		if h == 0 {
			h = 256
		}
		size := binary.LittleEndian.Uint32(appIconData[off+8 : off+12])
		offset := binary.LittleEndian.Uint32(appIconData[off+12 : off+16])
		entries[i] = iconEntry{w: w, h: h, offset: offset, size: size}
	}

	// Find best matches
	var bigBest, smallBest *iconEntry
	for i := range entries {
		e := &entries[i]
		if bigBest == nil || e.w > bigBest.w {
			bigBest = e
		}
		if smallBest == nil || e.w < smallBest.w {
			smallBest = e
		}
	}

	// Create and set icons
	if bigBest != nil && int(bigBest.offset)+int(bigBest.size) <= len(appIconData) {
		data := appIconData[bigBest.offset : bigBest.offset+bigBest.size]
		hicon := createIconFromData(data)
		if hicon != 0 {
			procSetIcon.Call(hwnd, WM_SETICON, ICON_BIG, hicon)
		}
	}
	if smallBest != nil && int(smallBest.offset)+int(smallBest.size) <= len(appIconData) {
		data := appIconData[smallBest.offset : smallBest.offset+smallBest.size]
		hicon := createIconFromData(data)
		if hicon != 0 {
			procSetIcon.Call(hwnd, WM_SETICON, ICON_SMALL, hicon)
		}
	}
}

// createIconFromData creates an HICON from raw icon resource data
// (BITMAPINFOHEADER + pixel data + AND mask) using CreateIconFromResourceEx.
func createIconFromData(data []byte) uintptr {
	if len(data) < 40 {
		return 0
	}
	r, _, _ := procCreateIcon.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		1, // fIcon = TRUE
		0x00030000, // version
		0, 0, // use default dimensions
		0, // LR_DEFAULTCOLOR
	)
	return r
}
