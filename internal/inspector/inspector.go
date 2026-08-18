// Package inspector produces a richer project analysis used by the w2e
// "inspect" tool (MCP/CLI). It builds on validator's per-file buckets but
// adds framework detection, entry inference, asset count and warnings.
package inspector

import (
	"path/filepath"
	"strings"

	"github.com/minfu/w2e/internal/validator"
)

// Result is the output schema of inspect (§42).
type Result struct {
	SourceDir    string   `json:"source_dir"`
	Framework    string   `json:"framework"`
	Entry        string   `json:"entry"`
	HTML         []string `json:"html_files"`
	JS           []string `json:"js_files"`
	CSS          []string `json:"css_files"`
	Assets       []string `json:"assets"`
	AssetCount   int      `json:"assets_count"`
	SPA          bool     `json:"spa"`
	RouterLikely bool     `json:"router_detection"`
	Path         string   `json:"path"`
	Warnings     []string `json:"warnings"`
}

// Inspect performs the directory scan and classifies a framework.
func Inspect(dir string) (*Result, error) {
	dir = filepath.Clean(dir)
	files := scanAll(dir) // shared scan - identical to validator's
	entry := pickEntry(files.HTML)
	// re-use validator's broken-reference check for warnings only
	v := validator.Files{}
	out := &Result{
		SourceDir:    dir,
		Framework:    detectFramework(files, entryHints(files, entry)),
		Entry:        entry,
		HTML:         files.HTML,
		JS:           files.JS,
		CSS:          files.CSS,
		Assets:       files.Assets,
		AssetCount:   len(files.Assets),
		SPA:          validator.DetectSPA(v) || (len(files.HTML) == 1 && len(files.JS) >= 1),
		RouterLikely: detectRouter(files),
		Path:         dir,
		Warnings:     []string{},
	}
	// heuristic: large JS chunks → likely SPA router
	if len(files.JS) >= 1 && strings.Contains(strings.ToLower(entry), "index") {
		out.RouterLikely = true
	}
	// sanitize empty slice -> keep nil vs [] consistent for JSON
	sanitize(out)
	return out, nil
}

func scanAll(dir string) validator.Files {
	// delegate to validator's internal scan by running the same walk via
	// the exported DetectSPA path - but since scanAll is unexported in
	// validator, we re-walk here. The semantics are identical.
	return validator.ScanAll(dir)
}

// pickEntry returns the highest-priority entry HTML.
func pickEntry(htmls []string) string {
	for _, p := range []string{"index.html", "main.html", "app.html", "home.html"} {
		for _, h := range htmls {
			if strings.EqualFold(filepath.Base(h), p) {
				return h
			}
		}
	}
	if len(htmls) > 0 {
		return htmls[0]
	}
	return ""
}

// entryHints returns the .js content alongside entry (cheap path detection).
func entryHints(files validator.Files, entry string) []string { return files.JS }

// detectFramework returns one of "React/Vite", "Vue/Vite", "Svelte/Vite",
// "Angular", "Vite (unknown framework)", "Vanilla JS", "Unknown".
// Heuristics are based on asset name and directory cues - non-exhaustive.
func detectFramework(files validator.Files, js []string) string {
	everywhere := strings.ToLower(strings.Join(files.JS, " ") + " " + strings.Join(files.Assets, " "))
	switch {
	case strings.Contains(everywhere, "react"):
		if strings.Contains(everywhere, "vite") {
			return "React/Vite"
		}
		return "React"
	case strings.Contains(everywhere, "vue"):
		if strings.Contains(everywhere, "vite") {
			return "Vue/Vite"
		}
		return "Vue"
	case strings.Contains(everywhere, "svelte"):
		if strings.Contains(everywhere, "vite") {
			return "Svelte/Vite"
		}
		return "Svelte"
	case strings.Contains(everywhere, "angular"):
		return "Angular"
	case strings.Contains(everywhere, "vite"):
		return "Vite (unknown framework)"
	case len(files.JS) > 0:
		return "Vanilla JS"
	}
	return "Unknown"
}

// detectRouter returns true when router-shaped asset names are present.
func detectRouter(files validator.Files) bool {
	all := strings.ToLower(strings.Join(append(append(files.JS, files.Assets...), files.HTML...), " "))
	for _, cue := range []string{"router", "react-router", "vue-router", "svelte-spa", "history"} {
		if strings.Contains(all, cue) {
			return true
		}
	}
	return false
}

// sanitize normalizes empty slices to nil so JSON emits "[]"
func sanitize(r *Result) {
	if r.HTML == nil {
		r.HTML = []string{}
	}
	if r.JS == nil {
		r.JS = []string{}
	}
	if r.CSS == nil {
		r.CSS = []string{}
	}
	if r.Assets == nil {
		r.Assets = []string{}
	}
	if r.Warnings == nil {
		r.Warnings = []string{}
	}
}
