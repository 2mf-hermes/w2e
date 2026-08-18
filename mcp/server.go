// Package mcp implements the w2e MCP server using mark3labs/mcp-go.
//
// It exposes the tools described in spec §40-§44:
//
//   - w2e_validate: validate a web project before packaging
//   - w2e_inspect:  analyze a web directory
//   - w2e_build:    build a Windows EXE from a web project (or online URL)
//   - w2e_doctor:   report host build-environment status
//   - w2e_version:  report w2e version information
//
// Transport: stdio (spec §39). The server reads JSON-RPC from stdin and
// writes JSON-RPC responses to stdout. Logging goes to stderr so MCP clients
// can capture it without corrupting the stdio protocol stream.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/minfu/w2e/internal/builder"
	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/inspector"
	"github.com/minfu/w2e/internal/validator"
	"github.com/minfu/w2e/internal/version"
)

// Stdio runs the MCP server on stdin/stdout. It blocks until stdin closes.
func Stdio(ctx context.Context) error {
	srv := server.NewMCPServer("w2e", version.Version, server.WithToolCapabilities(true))
	registerTools(srv, os.Stderr)
	stdio := server.NewStdioServer(srv)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}

func registerTools(s *server.MCPServer, log io.Writer) {
	s.AddTool(buildInspect(log))
	s.AddTool(buildValidate(log))
	s.AddTool(buildBuildTool(log))
	s.AddTool(buildDoctor(log))
	s.AddTool(buildVersion(log))
}

// --- shared helpers ------------------------------------------------------

func parseParams[Req any](args mcp.CallToolRequest) (*Req, error) {
	raw, err := json.Marshal(args.Params.Arguments)
	if err != nil {
		return nil, err
	}
	var req Req
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(text),
		},
	}
}

func jsonResult(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.NewTextContent(string(b))},
	}
}

// --- tools ---------------------------------------------------------------

// inspectRequest is the input schema of w2e_inspect (§42).
type inspectRequest struct {
	SourceDir string `json:"source_dir"`
}

func buildInspect(log io.Writer) (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("w2e_inspect",
		mcp.WithDescription(
			`Analyze a local web project directory and report its structure.

Use this tool before packaging to understand the project: which framework is in use (React/Vite, Vue/Vite, Svelte, Angular, Vanilla JS, etc.), how many HTML/CSS/JS/asset files there are, what the entry HTML file is, whether SPA routing is likely, and any warnings.

Input:
- source_dir (string, required): absolute or relative path to the directory to inspect (must contain at least one .html file).

Output:
JSON with: framework, entry, html_files, js_files, css_files, assets, assets_count, spa (bool), router_detection (bool), warnings.`),
		mcp.WithString("source_dir", mcp.Required(), mcp.Description("Path to the web project directory to inspect")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		r, err := parseParams[inspectRequest](req)
		if err != nil {
			return nil, fmt.Errorf("inspect: invalid request: %w", err)
		}
		if r.SourceDir == "" {
			return jsonResult(map[string]any{
				"success":    false,
				"error_code": string("INVALID_SOURCE"),
				"message":    "source_dir is required",
			}), nil
		}
		abs, err := filepath.Abs(r.SourceDir)
		if err != nil {
			return jsonResult(newErrEnvelope("INVALID_SOURCE",
				"could not resolve source_dir", err)), nil
		}
		st, err := os.Stat(abs)
		if err != nil || !st.IsDir() {
			return jsonResult(newErrEnvelope("INVALID_SOURCE",
				"source directory does not exist or is not a directory", nil)), nil
		}
		rep, err := inspector.Inspect(abs)
		if err != nil {
			return jsonResult(newErrEnvelope("INVALID_SOURCE",
				"inspect failed", err)), nil
		}
		return jsonResult(rep), nil
	}
}

// validateRequest is the input schema of w2e_validate (§40).
type validateRequest struct {
	SourceDir string `json:"source_dir"`
	Entry     string `json:"entry"`
}

func buildValidate(log io.Writer) (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("w2e_validate",
		mcp.WithDescription(
			`Validate a local web project before packaging it into a Windows EXE.

Use this tool to confirm the directory has an HTML entry point, can be served as a static site, and has no broken asset references.

Input:
- source_dir (string, required): path to the web project directory.
- entry (string, optional): override the entry HTML file; defaults to index.html or the best-prioritized match.

Output:
JSON with: source_dir, entry, results[], ready (bool), errors[], warnings[], and error_code on failure.`),
		mcp.WithString("source_dir", mcp.Required(), mcp.Description("Path to the web project directory to validate")),
		mcp.WithString("entry", mcp.Description("Optional override for the entry HTML file")),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		r, err := parseParams[validateRequest](req)
		if err != nil {
			return nil, fmt.Errorf("validate: invalid request: %w", err)
		}
		if r.SourceDir == "" {
			return jsonResult(newErrEnvelope("INVALID_SOURCE",
				"source_dir is required", nil)), nil
		}
		abs, err := filepath.Abs(r.SourceDir)
		if err != nil {
			return jsonResult(newErrEnvelope("INVALID_SOURCE",
				"could not resolve source_dir", err)), nil
		}
		cfg := &config.BuildConfig{SourceDir: abs, EntryFile: r.Entry}
		rep, err := validator.Validate(cfg)
		if err != nil {
			e := newErrEnvelope("INVALID_SOURCE", err.Error(), nil)
			return jsonResult(e), nil
		}
		return jsonResult(rep), nil
	}
}

// buildRequest is the input schema of w2e_build (§41).
type buildRequest struct {
	SourceDir   string `json:"source_dir"`
	Entry       string `json:"entry,omitempty"`
	Output      string `json:"output"`
	AppName     string `json:"app_name,omitempty"`
	WindowTitle string `json:"window_title,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Resizable   bool   `json:"resizable,omitempty"`
	Icon        string `json:"icon,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	Target      string `json:"target,omitempty"`
	KeepTemp    bool   `json:"keep_temp,omitempty"`
}

// toConfig copies buildRequest into a config.BuildConfig, applying spec §71
// defaults for omitted width / height / title.
func (r *buildRequest) toConfig() *config.BuildConfig {
	cfg := &config.BuildConfig{
		SourceDir:   r.SourceDir,
		EntryFile:   r.Entry,
		OutputFile:  r.Output,
		AppName:     r.AppName,
		WindowTitle: r.WindowTitle,
		Width:       r.Width,
		Height:      r.Height,
		Resizable:   r.Resizable,
		IconPath:    r.Icon,
		SourceURL:   r.SourceURL,
		Target:      r.Target,
	}
	if cfg.Width == 0 {
		cfg.Width = config.DefaultWidth
	}
	if cfg.Height == 0 {
		cfg.Height = config.DefaultHeight
	}
	if cfg.WindowTitle == "" {
		cfg.WindowTitle = config.DefaultWindowTitle
	}
	return cfg
}

func buildBuildTool(log io.Writer) (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("w2e_build",
		mcp.WithDescription(
			`Build a local HTML/CSS/JS/SPA web project into a standalone Windows executable.

Use this tool when the user wants to convert a web application into a Windows .exe. The web project is embedded into the EXE and launched inside WebView2 at runtime — the final user needs only the WebView2 Runtime installed on their Windows machine, nothing else.

Inputs:
- source_dir (string, required unless source_url is set): directory containing the web project (must contain an .html entry).
- source_url (string, optional): when set, the produced EXE loads this online URL directly instead of embedding local assets. Mutually exclusive with source_dir.
- entry (string, optional): override the entry HTML file.
- output (string, required): output EXE file path.
- app_name (string, optional): filesystem-safe application name; defaults to the output file name.
- window_title (string, optional): window title; defaults to "My App".
- width (int, optional): initial window width; defaults to 1024 (min 320).
- height (int, optional): initial window height; defaults to 720 (min 240).
- resizable (bool, optional): allow window resizing; defaults to true.
- icon (string, optional): path to an .ico or .png icon. If rcedit.exe is available on PATH, the icon is applied to the EXE; otherwise the default icon is used (the icon choice is recorded in w2e-icon.json beside the EXE).
- keep_temp (bool, optional, default false): keep the generated project on failure for debugging.

Output:
JSON with: success, output_path, app_name, size, duration_ms, warnings, and on failure: error_code, error_message.

The build invokes the Go toolchain, which must be installed on the host that runs w2e. The produced EXE does NOT require Go at runtime.

Path safety: output paths are normalized to absolute paths; paths inside C:\\Windows, C:\\Program Files, or any Windows system directory are rejected (§45).`),
		mcp.WithString("source_dir", mcp.Description("Path to the web project directory (required unless source_url is set)")),
		mcp.WithString("entry", mcp.Description("Optional entry HTML file override")),
		mcp.WithString("output", mcp.Required(), mcp.Description("Output executable file path")),
		mcp.WithString("target", mcp.Description("Target platform: windows, linux, darwin, or all. Defaults to windows.")),
		mcp.WithString("app_name", mcp.Description("Optional application name")),
		mcp.WithString("window_title", mcp.Description("Optional window title (default: My App)")),
		mcp.WithString("icon", mcp.Description("Optional .ico / .png icon path")),
		mcp.WithString("source_url", mcp.Description("Online URL mode: EXE loads this URL instead of embedding local assets")),
		mcp.WithNumber("width", mcp.Description("Initial window width in px (default 1024, min 320)")),
		mcp.WithNumber("height", mcp.Description("Initial window height in px (default 720, min 240)")),
		mcp.WithBoolean("resizable", mcp.Description("Whether the window can be resized (default true)")),
		mcp.WithBoolean("keep_temp", mcp.Description("Keep temp build directory on failure (default false)")),
	)

	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		r, err := parseParams[buildRequest](req)
		if err != nil {
			return nil, fmt.Errorf("build: invalid request: %w", err)
		}
		if r.SourceDir == "" && r.SourceURL == "" {
			return jsonResult(newErrEnvelope("INVALID_CONFIG",
				"source_dir or source_url is required", nil)), nil
		}
		if r.Output == "" {
			return jsonResult(newErrEnvelope("INVALID_CONFIG",
				"output is required", nil)), nil
		}
		cfg := r.toConfig()
		if err := cfg.Normalize(); err != nil {
			return jsonResult(newErrEnvelope("INVALID_CONFIG", err.Error(), nil)), nil
		}
		// Path safety (§45): reject traversal / system-directory output paths.
		abs, err := cfg.ResolveOutputAbs()
		if err != nil {
			return jsonResult(newErrEnvelope("OUTPUT_NOT_WRITABLE",
				"could not resolve output path", err)), nil
		}
		if pathError := rejectSystemPaths(abs); pathError != nil {
			return jsonResult(pathError), nil
		}

		eng := builder.New()
		eng.KeepTemp = r.KeepTemp
		res := eng.Build(cfg, nil)
		// MultiResult adapts: report only the per-target Result(s). If any
		// target failed, return the first failing one as an envelope so
		// clients get a useful error_code.
		if !res.Success {
			for _, t := range res.Targets {
				if !t.Success {
					return jsonResult(newErrEnvelope(t.ErrorCode, t.ErrorMessage, nil)), nil
				}
			}
		}
		return jsonResult(res), nil
	}
}

// rejectSystemPaths implements §45 path traversal / system-directory
// safeguards. Returns an *errEnvelope when the path is unsafe.
func rejectSystemPaths(abs string) any {
	cleanLow := strings.ToLower(filepath.Clean(abs))
	forbiddenPrefixes := []string{
		`c:\windows`, `c:\program files`, `c:\program files (x86)`,
		`c:\programdata`, `c:\windows\system32`,
	}
	for _, p := range forbiddenPrefixes {
		if cleanLow == p || strings.HasPrefix(cleanLow, p+`\`) {
			return newErrEnvelope("OUTPUT_NOT_WRITABLE",
				"refusing to write into a Windows system directory: "+abs, nil)
		}
	}
	if strings.Contains(cleanLow, `..`) {
		return newErrEnvelope("OUTPUT_NOT_WRITABLE",
			"relative path components are not allowed in output", nil)
	}
	return nil
}

// errEnvelope is the canonical agent-friendly error payload (§47).
type errEnvelope struct {
	Success    bool   `json:"success"`
	ErrorCode  string `json:"error_code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *errEnvelope) Error() string { return e.Message }

func newErrEnvelope(code, message string, err error) *errEnvelope {
	out := &errEnvelope{
		Success:   false,
		ErrorCode: code,
		Message:   message,
	}
	if err != nil {
		out.Message = message + ": " + err.Error()
	}
	if code == "ENTRY_NOT_FOUND" {
		out.Suggestion = "Specify another entry HTML file via --entry."
	}
	if code == "OUTPUT_NOT_WRITABLE" {
		out.Suggestion = "Choose an output path outside Windows system directories."
	}
	return out
}

// doctorResult is the output of w2e_doctor (§43).
type doctorResult struct {
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	Webview2Runtime   bool   `json:"webview2_runtime"`
	Permissions       bool   `json:"permissions"`
	Go                bool   `json:"go"`
	BuildAvailable    bool   `json:"build_available"`
	Webview2Source    string `json:"webview2_source,omitempty"`
}

func buildDoctor(log io.Writer) (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("w2e_doctor",
		mcp.WithDescription(
			`Check the current Windows build-environment for w2e.

Use this to confirm that w2e can produce Windows EXEs on this machine: requires Windows 10/11 x64 (or arm64), an installed Go toolchain, and ideally an installed WebView2 Runtime (the latter is **only** required on the target machine that runs the produced EXE, not on the build host).

Output:
JSON with: os, arch, webview2_runtime (bool), permissions (bool), go (bool), build_available (bool), webview2_source.`),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// builder.Doctor() belongs in builder pkg — wired below.
		out := builder.Doctor()
		return jsonResult(out), nil
	}
}

// --- version ------------------------------------------------------------

// versionResult is the output of w2e_version (§44).
type versionResult struct {
	Version         string `json:"version"`
	GoVersion       string `json:"go_version"`
	WebView2Version string `json:"webview2"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	BuildEngine     string `json:"build_engine_version"`
	MCPProtocol     string `json:"mcp_protocol"`
}

func buildVersion(log io.Writer) (mcp.Tool, server.ToolHandlerFunc) {
	t := mcp.NewTool("w2e_version",
		mcp.WithDescription(
			`Return w2e's version information.

Output:
JSON with: version, go_version, webview2, os, arch, build_engine_version, mcp_protocol.`),
	)
	return t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info := version.Get()
		v := versionResult{
			Version:         info.Version,
			GoVersion:       info.GoVersion,
			WebView2Version: info.WebView2Engine,
			OS:              info.OS,
			Arch:            info.Arch,
			BuildEngine:     info.BuildEngine,
			MCPProtocol:     info.MCPProtocol,
		}
		return jsonResult(v), nil
	}
}

// silence unused import safety
var _ = errors.New
