// Package validator — scanner.go performs a single-pass scan of a project
// directory and reports file buckets and cheap SPA heuristics.
package validator

import (
	"os"
	"path/filepath"
	"strings"
)

// Files holds per-type lists of relative asset file paths.
type Files struct {
	HTML   []string
	CSS    []string
	JS     []string
	Assets []string // images/fonts/misc binary & text assets
	SPA    bool     // true when the entry is a single root index + JS bundle
}

// ScanAll is the exported form of scanAll() so inspector can share the same
// single-pass walker without reimplementing it.
func ScanAll(dir string) Files { return scanAll(dir) }

// HasSPA reports whether we found heuristic signs of a SPA (router / single
// root index.html with a big JS bundle referenced).
func (f Files) HasSPA() bool { return len(f.HTML) == 1 && len(f.JS) >= 1 }

// scanAll walks dir once and groups files by extension.
func scanAll(dir string) Files {
	var f Files
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// skip hidden dirs (.git, node_modules explicitly)
			name := d.Name()
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(rel))
		switch ext {
		case ".html", ".htm":
			f.HTML = append(f.HTML, rel)
		case ".css":
			f.CSS = append(f.CSS, rel)
		case ".js", ".mjs", ".cjs":
			f.JS = append(f.JS, rel)
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico",
			".woff", ".woff2", ".ttf", ".eot", ".otf",
			".json", ".map", ".txt", ".wasm", ".mp4", ".webm":
			f.Assets = append(f.Assets, rel)
		}
		return nil
	})
	return f
}

// SPA detection: a project is "SPA-shaped" when the only HTML in scope is the
// entry index (no calendar/named HTML pages) and there are JS modules.
func detectSPA(f Files) bool { return len(f.HTML) == 1 && len(f.JS) >= 1 }

// Static alias for exported consumers.
func DetectSPA(f Files) bool { return detectSPA(f) }

// checkReferences is a light reference-resolver: parses the entry HTML for
// href/src targets present relative to the directory, then flags the ones
// that don't exist. It does NOT chase cross-file JS imports.
func checkReferences(dir, entry string, files Files) []string {
	data, err := os.ReadFile(filepath.Join(dir, entry))
	if err != nil {
		return nil
	}
	var broken []string
	src := string(data)
	// cheat-scan: extract bare quoted values from href="", src="".
	for _, marker := range []string{`href="`, `src="`, `href='`, `src='`} {
		for st := 0; ; {
			i := strings.Index(src[st:], marker)
			if i < 0 {
				break
			}
			i += st + len(marker)
			st = i
			var q byte = '"'
			if marker[len(marker)-1] == '\'' {
				q = '\''
			}
			j := strings.IndexByte(src[i:], q)
			if j < 0 {
				break
			}
			target := src[i : i+j]
			st = i + j
			if target == "" || strings.HasPrefix(target, "http") ||
				strings.HasPrefix(target, "data:") || strings.HasPrefix(target, "#") ||
				strings.HasPrefix(target, "javascript:") || strings.HasPrefix(target, "mailto:") ||
				strings.HasPrefix(target, "tel:") || strings.HasPrefix(target, "blob:") {
				continue
			}
			clean := strings.SplitN(target, "#", 2)[0]
			clean = strings.SplitN(clean, "?", 2)[0]
			if clean == "" {
				continue
			}
			if !FileExistsSafe(filepath.Join(dir, clean)) {
				broken = append(broken, clean)
			}
		}
	}
	return uniqueStrings(broken)
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
