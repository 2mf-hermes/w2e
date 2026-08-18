package builder

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed hostruntime/*
var hostRuntimeFS embed.FS

//go:embed host.exe
var hostExeBinary []byte

// hostMainArgs holds the inputs that main.go needs to specialize to the
// particular project being built.
type hostMainArgs struct {
	AppID       string
	WindowTitle string
	Width       int
	Height      int
	Resizable   string // "true" or "false"
	EntryFile   string
	RemoteURL   string
	ModeLocal   bool
	WebSubdir   string
	Debug       bool
}

// renderHostMainGo fills in the platform-specific template with the build
// args. The template is picked based on the target: Windows uses the
// pure-Go go-webview2 binding (CGO-free), while Linux/darwin use webview_go
// which relies on WebKit2GTK / WKWebView via CGO.
func renderHostMainGo(in hostMainArgs, target string) (string, error) {
	tmplName := "hostruntime/main_windows.go.txt"
	switch target {
	case "linux", "darwin":
		tmplName = "hostruntime/main_unix.go.txt"
	}
	tmpl, err := hostRuntimeFS.ReadFile(tmplName)
	if err != nil {
		return "", fmt.Errorf("read host template: %w", err)
	}
	out := string(tmpl)
	out = strings.ReplaceAll(out, "{{APP_ID}}", in.AppID)
	out = strings.ReplaceAll(out, "{{WINDOW_TITLE}}", escapeString(in.WindowTitle))
	out = strings.ReplaceAll(out, "{{WIDTH}}", fmt.Sprintf("%d", in.Width))
	out = strings.ReplaceAll(out, "{{HEIGHT}}", fmt.Sprintf("%d", in.Height))
	out = strings.ReplaceAll(out, "{{RESIZABLE}}", in.Resizable)
	out = strings.ReplaceAll(out, "{{ENTRY_FILE}}", escapeString(in.EntryFile))
	out = strings.ReplaceAll(out, "{{REMOTE_URL}}", escapeString(in.RemoteURL))
	out = strings.ReplaceAll(out, "{{MODE_LOCAL}}", fmt.Sprintf("%v", in.ModeLocal))
	out = strings.ReplaceAll(out, "{{WEB_SUBDIR}}", in.WebSubdir)
	out = strings.ReplaceAll(out, "{{DEBUG}}", fmt.Sprintf("%v", in.Debug))
	return out, nil
}

// escapeString produces a Go double-quoted string literal for arbitrary s.
// It escapes newlines, double quotes, and backslashes to avoid breaking the
// generated code when titles contain special characters.
func escapeString(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
