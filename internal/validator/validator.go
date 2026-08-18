// Package validator performs static checks on a web project directory before
// packaging into a Windows EXE. It is decoupled from the builder, GUI, CLI
// and MCP, all of which call into this and inspector.
package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/errcode"
)

// Status is a per-check verdict.
type Status string

const (
	StatusOK    Status = "ok"
	StatusWarn  Status = "warn"
	StatusFail  Status = "fail"
	StatusSkip  Status = "skipped"
)

// Check names the analyzer step that produced a Result.
type Check string

const (
	CheckEntry       Check = "entry"
	CheckHTML        Check = "html"
	CheckCSS         Check = "css"
	CheckJS          Check = "js"
	CheckAssets      Check = "assets"
	CheckSPA         Check = "spa"
	CheckMIME        Check = "mime_types"
	CheckReferences  Check = "references"
)

// Result is one check's outcome.
type Result struct {
	Check     Check   `json:"check"`
	Status    Status  `json:"status"`
	Message   string  `json:"message"`
	Sensitive string  `json:"-"`
}

// Report aggregates all checks for a project.
type Report struct {
	SourceDir  string   `json:"source_dir"`
	Entry      string   `json:"entry"`
	Results    []Result `json:"results"`
	Ready      bool     `json:"ready"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	ErrorCode  string   `json:"error_code,omitempty"`
}

// Validate runs the project checks described in spec §20, §32. It returns a
// human-actionable Report and codes any fatal problem with an *errcode.Error
// attached via result ErrorCode.
func Validate(cfg *config.BuildConfig) (*Report, error) {
	if cfg == nil {
		return nil, errcode.New(errcode.InvalidConfig,
			"build config is nil", "provide a BuildConfig", nil)
	}
	if !cfg.HasSourceDir() {
		return nil, errcode.New(errcode.InvalidSource,
			"source_dir is required", "specify a directory containing index.html", nil)
	}
	abs, err := cfg.ResolveSourceAbs()
	if err != nil {
		return nil, errcode.New(errcode.InvalidSource,
			"could not resolve source dir", "use an absolute or local path", err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, errcode.New(errcode.InvalidSource,
			"source directory not found", "check the path and try again", err)
	}
	if !st.IsDir() {
		return nil, errcode.New(errcode.InvalidSource,
			"source path is not a directory", "w2e needs a folder, not a file", nil)
	}

	r := &Report{SourceDir: abs, Entry: cfg.EntryFile}
	var f Files
	defer func() { _ = f }()
	defer func() {
		if len(r.Errors) == 0 {
			r.Ready = true
		}
	}()

	{ // entry
		entry, res := resolveEntry(abs, cfg.EntryFile)
		r.Entry = entry
		cfg.EntryFile = entry // normalize the caller's config in-place too
		if res.Status == StatusFail {
			r.Errors = append(r.Errors, res.Message)
			r.ErrorCode = string(errcode.EntryNotFound)
			r.Results = append(r.Results, res)
			return r, errcode.New(errcode.EntryNotFound, res.Message,
				"verify that index.html exists or pass --entry", nil)
		}
		r.Results = append(r.Results, res)
	}

	all := scanAll(abs)
	r.Results = append(r.Results,
		resultOK(CheckHTML, fmt.Sprintf("found %d HTML file(s)", len(all.HTML))),
		maybeAsset(CheckCSS, len(all.CSS)),
		maybeAsset(CheckJS, len(all.JS)),
		maybeAsset(CheckAssets, len(all.Assets)),
	)
	f.SPA = detectSPA(all)
	if f.SPA {
		r.Results = append(r.Results, resultOK(CheckSPA, "SPA routing support detected"))
	} else {
		r.Results = append(r.Results, Result{
			Check: CheckSPA, Status: StatusWarn,
			Message: "no SPA router detected; client-side routes may break — w2e supports index.html fallback regardless",
		})
	}
	r.Results = append(r.Results, resultOK(CheckMIME, mimeSummary()))

	// broken references: check <a href>/<link>/<script src>/<img src> all reachable
	broken := checkReferences(abs, r.Entry, all)
	if len(broken) == 0 {
		r.Results = append(r.Results, resultOK(CheckReferences, "all referenced local assets resolve"))
	} else {
		r.Results = append(r.Results, Result{
			Check: CheckReferences, Status: StatusWarn,
			Message: fmt.Sprintf("%d broken asset reference(s): %s", len(broken), strings.Join(broken[:minN(3, len(broken))], ", ")),
		})
		for _, b := range broken {
			r.Warnings = append(r.Warnings, "broken reference: "+b)
		}
	}

	return r, nil
}

func minN(a, b int) int { if a < b { return a }; return b }

func resultOK(c Check, m string) Result { return Result{Check: c, Status: StatusOK, Message: m} }

func maybeAsset(c Check, n int) Result {
	if n == 0 {
		return Result{Check: c, Status: StatusWarn,
			Message: fmt.Sprintf("no %s files detected", c)}
	}
	return resultOK(c, fmt.Sprintf("found %d file(s)", n))
}

// resolveEntry picks the entry HTML according to spec §19 priority:
// index.html > main.html > app.html > home.html; otherwise first .html found.
func resolveEntry(dir, hint string) (string, Result) {
	if hint != "" {
		full := filepath.Join(dir, hint)
		if !FileExistsSafe(full) {
			return hint, Result{Check: CheckEntry, Status: StatusFail,
				Message: fmt.Sprintf("specified entry not found: %s", hint)}
		}
		return hint, resultOK(CheckEntry, "entry: "+hint)
	}
	priority := []string{"index.html", "main.html", "app.html", "home.html"}
	for _, p := range priority {
		if FileExistsSafe(filepath.Join(dir, p)) {
			return p, resultOK(CheckEntry, "entry: "+p)
		}
	}
	// fallback: first .html in top-level dir
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".html") {
			return e.Name(), resultOK(CheckEntry, "entry: "+e.Name())
		}
	}
	return "", Result{Check: CheckEntry, Status: StatusFail,
		Message: "no HTML entry (index.html/main.html/app.html/home.html) found"}
}

// FileExistsSafe returns whether path exists and is not a directory.
func FileExistsSafe(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func mimeSummary() string {
	return "built-in MIME table covers html/css/js/mjs/json/svg/png/jpg/jpeg/gif/webp/ico/woff/woff2/ttf"
}
