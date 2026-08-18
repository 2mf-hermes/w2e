// Package browser opens a URL in the system default browser on Windows.
//
// We use only the Windows shell "start" verb via exec.Command, which doesn't
// require Go's CGO, and doesn't require elevation. Intermediate URL schemes
// are re-checked here to avoid launching exotic things like file:// paths.
package browser

import (
	"errors"
	"os/exec"
	"strings"
)

// Open launches u in the user's default browser. It refuses to run for
// schemes other than http and https.
func Open(u string) error {
	if u == "" {
		return errors.New("browser: empty URL")
	}
	ul := strings.ToLower(strings.TrimSpace(u))
	if !strings.HasPrefix(ul, "http://") && !strings.HasPrefix(ul, "https://") {
		return errors.New("browser: refused non-http(s) URL")
	}
	// `rundll32 url.dll,FileProtocolHandler <url>` works on all Win10/11
	// without spawning a console window.
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u)
	return cmd.Start()
}
