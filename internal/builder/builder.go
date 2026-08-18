// Package builder is the build pipeline that turns a web project into a
// real native executable on Windows, Linux, and macOS.
//
// Design (spec §35, §37):
//   1. validate the source via the validator package
//   2. materialize a temp Go project whose main.go is the self-contained
//      host runtime (TEMPer) that:
//        - embeds the user's web assets via //go:embed all:web
//        - launches a localhost HTTP server on 127.0.0.1:0
//        - opens a webview window pointed at the server
//        - exits cleanly when the window closes
//   3. run `go build` (with platform-appropriate GOOS/GOARCH/CGO) to produce
//      the target executable (PE / ELF / Mach-O)
//   4. verify the output format
//   5. move/copy to the requested output
//
// Each pipeline step emits via a Progress callback so GUI/CLI can show stages.
//
// CROSS-PLATFORM COMPILE NOTES:
//   - windows → CGO_ENABLED=0 (pure Go, go-webview2-style fallback to webview_go)
//   - linux   → CGO_ENABLED=1, requires gcc + webkit2gtk dev headers
//   - darwin  → CGO_ENABLED=1, requires Xcode/clang + WebKit (ships with macOS)
//   - Cross-compiling from Windows to linux/darwin requires a matching C
//     cross-compiler (e.g. x86_64-linux-gnu-gcc, osxcross). See README.
package builder

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/errcode"
	"github.com/minfu/w2e/internal/icon"
	"github.com/minfu/w2e/internal/logging"
	"github.com/minfu/w2e/internal/platform"
	"github.com/minfu/w2e/internal/validator"
)

// Progress is called with each pipeline step's status; status is
// "start"|"ok"|"warn"|"fail" and msg/percent are advisory.
type Progress func(step, status, msg string, percent int)

// Result is the standard BuildResult (spec §38).
type Result struct {
	Success      bool     `json:"success"`
	OutputPath   string   `json:"output_path,omitempty"`
	AppName      string   `json:"app_name,omitempty"`
	Target       string   `json:"target,omitempty"`
	Format       string   `json:"format,omitempty"`
	Size         int64    `json:"size,omitempty"`
	DurationMs   int64    `json:"duration_ms,omitempty"`
	ErrorCode    string   `json:"error_code,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// MultiResult holds the outcomes of a single Build() call expanded across one
// or more targets. When Target is a single platform, Results has one entry.
// When Target == "all", Results has up to three entries (per platform).
type MultiResult struct {
	Targets []Result `json:"targets"`
	Success bool     `json:"success"`
}

// Engine is the single build coordinator shared by GUI / CLI / MCP.
type Engine struct {
	Log      *logging.Logger
	GoPath   string // override; defaults to "go" on PATH
	KeepTemp bool   // keep temp on failure (§55)
	TempRoot string // optional override for tests
}

// New returns an Engine with INFO-level logging.
func New() *Engine { return &Engine{Log: logging.Default(logging.LevelInfo, "builder")} }

// Build runs the full pipeline (§37) and returns a MultiResult covering all
// requested targets. progress may be nil.
func (e *Engine) Build(cfg *config.BuildConfig, progress Progress) MultiResult {
	if progress == nil {
		progress = func(_, _, _ string, _ int) {}
	}
	if err := cfg.Normalize(); err != nil {
		return MultiResult{Targets: []Result{failResult(errcode.InvalidConfig, err.Error(), time.Now())}, Success: false}
	}

	target, perr := platform.Parse(cfg.Target)
	if perr != nil {
		return MultiResult{Targets: []Result{failResult(errcode.InvalidConfig, perr.Error(), time.Now())}, Success: false}
	}
	targets := platform.Expand(target)

	// Validate source once (shared across all targets).
	if cfg.HasSourceDir() {
		progress("validate", "start", "validating source", 5)
		rep, err := validator.Validate(cfg)
		if err != nil {
			return MultiResult{Targets: []Result{failResult(errcode.InvalidSource, err.Error(), time.Now())}, Success: false}
		}
		progress("validate", "ok", "project validated (entry: "+rep.Entry+")", 10)
	}

	results := make([]Result, 0, len(targets))
	anySuccess := false
	for i, t := range targets {
	idx := i + 1
		total := len(targets)
		// Per-target progress prefix in multi-target mode.
		pprog := progress
		if total > 1 {
			pprog = func(step, status, msg string, percent int) {
				// Scale percent into this target's slice.
				sliceStart := (idx - 1) * 100 / total
				scaled := sliceStart + percent/total
				progress(step, status, fmt.Sprintf("[%s] %s", t, msg), scaled)
			}
		}
		r := e.buildSingle(cfg, t, pprog)
		results = append(results, r)
		if r.Success {
			anySuccess = true
		}
	}
	return MultiResult{Targets: results, Success: anySuccess}
}

// buildSingle runs the full pipeline for one specific target platform.
func (e *Engine) buildSingle(cfg *config.BuildConfig, t platform.Target, progress Progress) Result {
	start := time.Now()
	if progress == nil {
		progress = func(_, _, _ string, _ int) {}
	}

	// Windows: use pre-compiled host template (no Go compiler needed at runtime).
	if t == platform.TargetWindows {
		return e.buildFromTemplate(cfg, progress)
	}

	// Linux/macOS: require Go compiler + C toolchain (CGO needed).
	if e.GoPath == "" {
		e.GoPath = "go"
	}

	tmpRoot := e.TempRoot
	if tmpRoot == "" {
		tmpRoot = filepath.Join(os.TempDir(), "w2e")
	}
	_ = os.MkdirAll(tmpRoot, 0o755)
	tmp, err := os.MkdirTemp(tmpRoot, "build-"+string(t)+"-")
	if err != nil {
		return failResult(errcode.BuildFailed, "could not create temp build dir", start)
	}
	if !e.KeepTemp {
		defer func() { _ = os.RemoveAll(tmp) }()
	}

	progress("prepare", "start", "preparing host project", 15)
	projRoot := filepath.Join(tmp, "host")
	if err := os.MkdirAll(projRoot, 0o755); err != nil {
		return failResult(errcode.BuildFailed, "could not create project dir", start)
	}

	progress("embed", "start", "embedding web assets", 20)
	if err := e.materializeHost(projRoot, cfg, t); err != nil {
		return failResult(errcode.BuildFailed, "could not materialize host: "+err.Error(), start)
	}
	progress("embed", "ok", "host project scaffolded", 25)

	// Output path is target-suffixed: MyApp.exe (windows), MyApp (linux/darwin).
	// For "all", append the target name to avoid collisions.
	outAbs, err := cfg.ResolveOutputAbs()
	if err != nil {
		return failResult(errcode.OutputNotWritable, "could not resolve output path", start)
	}
	outAbs = applyTargetSuffix(outAbs, cfg.Target, t)
	if err := ensureOutputDir(outAbs); err != nil {
		return failResult(errcode.OutputNotWritable, "output directory not writable", start)
	}

	// Cross-compile pre-check.
	cross := string(t) != runtime.GOOS
	if t.NeedsCGO() {
		// Check C compiler availability for the target.
		if ccerr := e.checkCCompiler(t, cross); ccerr != nil {
			return failResult(errcode.NativeToolchain, ccerr.Error(), start)
		}
	}

	progress("compile", "start", fmt.Sprintf("compiling %s binary", t.Format()), 30)
	// generate go.sum for the generated host project first.
	tidyCmd := exec.Command(e.GoPath, "mod", "tidy")
	tidyCmd.Dir = projRoot
	tidyCmd.Env = buildEnv(t)
	if tOut, terr := tidyCmd.CombinedOutput(); terr != nil {
		return failResult(errcode.BuildFailed,
			"go mod tidy failed: "+strings.TrimSpace(string(tOut))+" / "+terr.Error(), start)
	}

	binName := "app" + t.Suffix()
	binOut := filepath.Join(projRoot, binName)

	args := []string{"build"}
	if t == platform.TargetWindows {
		args = append(args, "-ldflags", "-H windowsgui")
	}
	args = append(args, "-o", binOut, ".")
	buildCmd := exec.Command(e.GoPath, args...)
	buildCmd.Dir = projRoot
	buildCmd.Env = buildEnv(t)
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		code := errcode.BuildFailed
		if cross {
			code = errcode.CrossCompileFailed
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		} else {
			msg = msg + "\n" + err.Error()
		}
		return failResult(code, "go build failed: "+msg, start)
	}
	progress("compile", "ok", "binary compiled", 75)

	progress("verify", "start", "verifying "+t.Format(), 80)
	if verr := verifyFormat(t, binOut); verr != nil {
		return failResult(errcode.VerifyFailed, verr.Error(), start)
	}
	progress("verify", "ok", t.Format()+" verification passed", 88)

	if cfg.IconPath != "" && t == platform.TargetWindows {
		ref, _, perr := icon.Prepare(cfg.IconPath)
		if perr != nil {
			progress("icon", "warn", "icon invalid: "+perr.Error(), 90)
		} else if ref.ICOPath != "" {
			ref, _ = icon.Apply(binOut, ref)
			_ = icon.Persist(filepath.Dir(outAbs), ref)
			if ref.Applied {
				progress("icon", "ok", "icon applied via rcedit", 92)
			} else {
				progress("icon", "warn", "rcedit unavailable; default icon used (recorded in w2e-icon.json)", 92)
			}
		}
	}

	progress("output", "start", "moving binary to output", 92)
	if err := copyFile(binOut, outAbs); err != nil {
		return failResult(errcode.OutputNotWritable,
			"could not copy binary to output (path in use or read-only?)", start)
	}
	st, _ := os.Stat(outAbs)
	if st == nil {
		return failResult(errcode.VerifyFailed, "output missing after copy", start)
	}
	progress("output", "ok", t.Format()+" at "+outAbs, 100)

	return Result{
		Success:    true,
		OutputPath: outAbs,
		AppName:    cfg.AppName,
		Target:     string(t),
		Format:     t.Format(),
		Size:       st.Size(),
		DurationMs: time.Since(start).Milliseconds(),
		Warnings:   []string{},
	}
}

// buildEnv returns the environment for `go build` / `go mod tidy` targeting t.
func buildEnv(t platform.Target) []string {
	env := os.Environ()
	set := func(k, v string) {
		// Remove any existing entry, then append.
		prefix := k + "="
		out := env[:0]
		for _, e := range env {
			if !strings.HasPrefix(e, prefix) {
				out = append(out, e)
			}
		}
		env = append(out, k+"="+v)
	}

	set("GOOS", t.GOOS())
	set("GOARCH", t.GOARCH())
	if t.NeedsCGO() {
		set("CGO_ENABLED", "1")
		// On cross builds, CC must point at a matching cross compiler.
		switch t {
		case platform.TargetLinux:
			if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
				set("CC", "x86_64-linux-gnu-gcc")
			}
		case platform.TargetDarwin:
			if runtime.GOOS != "darwin" {
				set("CC", "o64-clang")
			}
		}
	} else {
		set("CGO_ENABLED", "0")
	}
	return env
}

// checkCCompiler probes for a C compiler suitable for the target.
// Returns nil if one looks present, an error describing the missing toolchain
// otherwise. This is a best-effort pre-check — the real compile will surface
// the precise error if this check passes but the compile later fails.
func (e *Engine) checkCCompiler(t platform.Target, cross bool) error {
	cc := "gcc"
	if cross {
		switch t {
		case platform.TargetLinux:
			cc = "x86_64-linux-gnu-gcc"
		case platform.TargetDarwin:
			cc = "o64-clang"
		}
	}
	if t == platform.TargetDarwin && runtime.GOOS == "darwin" {
		cc = "clang"
	}
	if _, err := exec.LookPath(cc); err != nil {
		return fmt.Errorf("C compiler %q not on PATH (required for %s). On Linux install gcc + libwebkit2gtk-4.1-dev; on macOS install Xcode command-line tools; for cross-compile see README", cc, t.Format())
	}
	return nil
}

// verifyFormat dispatches to the per-format verifier.
func verifyFormat(t platform.Target, p string) error {
	switch t {
	case platform.TargetWindows:
		return VerifyPE(p)
	case platform.TargetLinux:
		return VerifyELF(p)
	case platform.TargetDarwin:
		return VerifyMachO(p)
	}
	return fmt.Errorf("unknown target: %s", t)
}

// applyTargetSuffix rewrites the output path for a specific target:
//   "all" mode: MyApp.exe → MyApp-windows.exe, MyApp-linux, MyApp-darwin
//   single mode: keep as-is (the user chose the platform explicitly).
func applyTargetSuffix(outAbs, originalTarget string, t platform.Target) string {
	if originalTarget != "all" {
		// Single target requested—keep the user-specified name. But if the
		// user left a windows-only ".exe" suffix while targeting non-windows,
		// swap it for platform-appropriate suffix.
		return fixSuffixFor(outAbs, t)
	}
	ext := filepath.Ext(outAbs)
	base := strings.TrimSuffix(outAbs, ext)
	return base + "-" + string(t) + t.Suffix()
}

// fixSuffixFor ensures the output path's extension matches the target.
func fixSuffixFor(outAbs string, t platform.Target) string {
	ext := filepath.Ext(outAbs)
	if t == platform.TargetWindows {
		if ext != ".exe" {
			return outAbs + ".exe"
		}
		return outAbs
	}
	// Linux/macOS: strip any ".exe" suffix.
	if ext == ".exe" {
		return strings.TrimSuffix(outAbs, ext)
	}
	return outAbs
}

func failResult(code errcode.Code, msg string, start time.Time) Result {
	return Result{
		Success:      false,
		ErrorCode:    string(code),
		ErrorMessage: msg,
		DurationMs:   time.Since(start).Milliseconds(),
		Warnings:     []string{},
	}
}

func (e *Engine) fail(code errcode.Code, msg string, start time.Time) Result {
	e.Log.Errorf("build failed: %s — %s", code, msg)
	return failResult(code, msg, start)
}

// materializeHost props up the generated project tree.
func (e *Engine) materializeHost(projRoot string, cfg *config.BuildConfig, t platform.Target) error {
	// web content subdir (only populated in local mode)
	webSubdir := "web"
	if cfg.HasSourceDir() {
		webDir := filepath.Join(projRoot, webSubdir)
		if err := os.MkdirAll(webDir, 0o755); err != nil {
			return err
		}
		srcAbs, _ := cfg.ResolveSourceAbs()
		if err := copyTree(srcAbs, webDir); err != nil {
			return err
		}
	}

	// Write main.go via the template.
	resizableStr := "true"
	if !cfg.Resizable {
		resizableStr = "false"
	}
	entry := cfg.EntryFile
	if entry == "" {
		entry = "index.html"
	}
	modeLocal := cfg.HasSourceDir()
	appID := sanitizeAppID(cfg.AppName)
	if appID == "" {
		appID = "w2eapp"
	}

	mainBody, err := renderHostMainGo(hostMainArgs{
		AppID:       appID,
		WindowTitle: cfg.WindowTitle,
		Width:       cfg.Width,
		Height:      cfg.Height,
		Resizable:   resizableStr,
		EntryFile:   entry,
		RemoteURL:   cfg.SourceURL,
		ModeLocal:   modeLocal,
		WebSubdir:   webSubdir,
		Debug:       cfg.Debug,
	}, string(t))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projRoot, "main.go"), []byte(mainBody), 0o644); err != nil {
		return err
	}

	// Per-target go.mod dependency:
	//   windows → pure-Go go-webview2 (no CGO dependency at any layer)
	//   linux/darwin → webview/webview_go (needs CGO + native WebKit)
	// Use the current Go version so the build always succeeds locally.
	// runtime.Version() returns "go1.X.Y" but go.mod requires "go X.Y" (with
	// a space, and optionally a patch).  We strip the prefix and rebuild.
	raw := runtime.Version() // e.g. "go1.25.13"
	goVer := strings.TrimPrefix(raw, "go") // "1.25.13"
	if goVer == "" || goVer == raw {
		goVer = "1.22"
	}
	// go.mod go directive accepts "go X" or "go X.Y" or "go X.Y.Z".
	// Cap at major.minor only for maximum compatibility.
	parts := strings.SplitN(goVer, ".", 3)
	if len(parts) >= 2 {
		goVer = parts[0] + "." + parts[1]
	}
	var goMod string
	if t == platform.TargetWindows {
		goMod = "module github.com/w2e/host\n\ngo " + goVer + "\n\nrequire github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808\n"
	} else {
		goMod = "module github.com/w2e/host\n\ngo " + goVer + "\n\nrequire github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6\n"
	}
	return os.WriteFile(filepath.Join(projRoot, "go.mod"), []byte(goMod), 0o644)
}

func ensureOutputDir(outAbs string) error {
	dir := filepath.Dir(outAbs)
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("output path parent is not a directory: %s", dir)
	}
	t := filepath.Join(dir, ".w2e_write_test")
	if err := os.WriteFile(t, []byte("x"), 0o644); err != nil {
		return err
	}
	_ = os.Remove(t)
	return nil
}

func copyFile(src, dst string) error {
	// Remove destination first to avoid sharing violations on Windows.
	os.Remove(dst)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// copyTree copies the directory at src to dst recursively, preserving
// relative paths. node_modules and dot-folders are skipped (spec §10).
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || (strings.HasPrefix(name, ".") && name != "." && name != "..") {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

// sanitizeAppID strips characters that are unsafe for filesystem paths.
func sanitizeAppID(s string) string {
	out := strings.Builder{}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-' || r == '_':
			out.WriteRune(r)
		}
	}
	if out.Len() == 0 {
		return "w2eapp"
	}
	return out.String()
}
