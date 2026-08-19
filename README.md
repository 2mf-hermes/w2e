<p align="center">
  <b>🌐 Read this page in your language:</b><br>
  <a href="README.md">English</a> · <a href="README.zh-TW.md">繁體中文</a> · <a href="README.zh-CN.md">简体中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a>
</p>

# w2e — Web → Native EXE packager (Windows · Linux · macOS)

w2e turns any HTML/CSS/JS/SPA web project into a **standalone native
executable** on Windows, Linux, and macOS — one web project, three targets.
Each produced binary launches the user's system webview (Edge WebView2 /
WebKit2GTK / WKWebView) pointed at an embedded `127.0.0.1:0` localhost
server, so no console window, no Admin/UAC, and no Node.js/Python runtime is
required on the user's machine.

![w2e Screenshot](screenshot/screenshot.png)

It ships as two products from one codebase:

| Product | What it does |
| --- | --- |
| **w2e Desktop** (`w2e.exe` / `w2e` / `w2e.app`) | Apple / iOS 26 "Liquid Glass" GUI. Pick a project, adjust the window, click **打包**, get one or all three binaries. |
| **w2e MCP Server** (`w2e mcp`, or `w2e-mcp.exe`) | An MCP stdio server exposing 5 tools so AI agents (Claude Code, Codex, Gemini CLI, …) can validate / inspect / build / doctor / version programmatically. |

---

## Quick start (Windows host)

```powershell
# Build w2e (requires Go 1.23+, downloads 1.25 toolchain automatically)
# Use build.ps1 so w2e.exe links with subsystem=GUI (no DOS console window):
.\build.ps1
.\build.ps1 -AlsoMcp          # also builds bin\w2e-mcp.exe
# (Equivalent manual command:)
# go build -ldflags "-X github.com/minfu/w2e/internal/version.Version=1.0.0-dev -H windowsgui" -o bin\w2e.exe .\cmd\w2e
# go build -o bin\w2e-mcp.exe .\cmd\w2e-mcp

# Package a web project into a Windows EXE (~6s on a modern machine)
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp.exe --title "My App"

# Cross-compile all three platforms in a single call:
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp --target all

# Or launch the GUI:
.\bin\w2e.exe
```

Outputs:

| target | file | format |
| --- | --- | --- |
| `windows` | `MyApp.exe` (or `MyApp-windows.exe` in `--target all`) | PE, GUI subsystem, no console window |
| `linux`   | `MyApp` (or `MyApp-linux`) | ELF 64-bit |
| `darwin`  | `MyApp` (or `MyApp-darwin`) | Mach-O 64-bit |

On every target, the end user only needs the **platform's webview runtime**
installed — Edge WebView2 Runtime on Windows (bundled with Windows 11),
WebKit2GTK on Linux (`libwebkit2gtk-4.1` ships with most desktop distros),
and WKWebView on macOS (built-in). No Go toolchain is required at runtime.

---

## System requirements

| Role | Requirement |
| --- | --- |
| **Windows build host** | Windows 10/11 x64/ARM64, **Go 1.23+**. Cross-compile to Linux/macOS additionally requires a matching C cross-compiler (see below). |
| **Linux build host** | Any modern Linux, **Go 1.23+**, `gcc`, and `libwebkit2gtk-4.1-dev` (+ `libgtk-3-dev`). Native Linux builds only. |
| **macOS build host** | macOS 10.13+, **Go 1.23+**, Xcode command-line tools (`clang`). Native macOS builds only. |
| **End user** | The platform's webview runtime (WebView2 / WebKit2GTK / WKWebView). |

Check your environment with `w2e doctor`:

```text
w2e doctor — environment diagnostics

  Windows x64:         ✓
  Go toolchain:        ✓
  WebView2 Runtime:    ✕ (not installed)   <-- only required at runtime
  Temporary Directory: ✓ C:\Users\you\AppData\Local\Temp
  User data folder:    ✓

Build targets:
  windows:              ✓ native (Go=true / C=true)
  linux:                ✕ cross-compile (Go=true / C=false)
    missing cross C compiler: install the matching toolchain (see README)
  darwin:               ✕ cross-compile (Go=true / C=false)
    missing cross C compiler: install the matching toolchain (see README)

build_available: true
cross_compile_available: false
```

> **Cross-compilation.** Windows→Windows is pure Go (`CGO_ENABLED=0`) and
> needs nothing but the Go toolchain. Building Linux/macOS binaries requires
> `CGO_ENABLED=1` plus a C compiler for the target. On native hosts this is
> just `gcc` (Linux) or `clang` (macOS, via `xcode-select --install`). To
> cross-compile from Windows, install:
>
> | target | C compiler | kit |
> | --- | --- | --- |
> | linux/amd64 from Windows/macOS | `x86_64-linux-gnu-gcc` | `mingw-w64` cross package / Zig ccache / Docker |
> | darwin/amd64 from Linux/Windows | `o64-clang` | `osxcross` (needs a macOS SDK) |
>
> For the easiest reliable cross builds, run `w2e build --target all` on each
> native host (CI matrix). One-EXE deployments stay pure Go on Windows.

---

## CLI reference

```text
w2e                      Launch the GUI
w2e build SOURCE [flags] Build a native executable from a web project
  --entry FILE             Override the entry HTML file (default: detected)
  --name NAME              Application name (default: output basename)
  --title TITLE            Window title         (default: "My App")
  --width N                Initial window width (default: 1024, min: 320)
  --height N               Initial window height(default: 720, min: 240)
  --icon PATH              Optional .ico or .png icon path
  --no-resizable           Disable window resizing
  --output PATH            Output path (required; suffixed per target in --target all)
  --keep-temp              Keep temp build dir on failure
  --url URL                Online-URL mode: binary loads this URL instead of SOURCE_DIR
  --target PLATFORM        windows | linux | darwin | all (default: windows)
w2e validate SOURCE     Validate a web project before building
w2e doctor              Diagnose the local build environment (incl. cross targets)
w2e mcp                 Start the MCP server (stdio transport)
w2e version             Print version information
w2e help                Show this help
```

### Online-URL mode

`w2e build --url https://...` produces a binary that simply loads the
given URL directly inside the platform webview — useful for packaging a
hosted app as a desktop launcher.

---

## MCP integration

`w2e mcp` (or the dedicated `w2e-mcp.exe` binary) speaks the Model Context
Protocol over stdio. Wire it into any agent that supports MCP stdio servers.
Example Claude Code / Gemini CLI config:

```jsonc
{
  "mcpServers": {
    "w2e": {
      "command": "C:\\path\\to\\bin\\w2e-mcp.exe"
    }
  }
}
```

### Tools exposed

| Tool | Purpose |
| --- | --- |
| `w2e_validate` | Validate a local web project before packaging |
| `w2e_inspect` | Analyze a directory (framework, entry, counts, SPA, warnings) |
| `w2e_build` | Build a local web project — or a `source_url` — into an EXE |
| `w2e_doctor` | Report host build-environment status |
| `w2e_version` | Report w2e version information |

All responses are JSON. Failures include an `error_code` and a machine-readable
`message`, plus a `suggestion` when relevant.

### Security

`w2e_build` **rejects** output paths inside Windows system directories
(`C:\Windows`, `C:\Program Files`, `C:\Program Files (x86)`, `C:\ProgramData`,
`C:\Windows\System32`) and refuses relative `..` traversal in the output path.
Source directories are constrained to paths the agent has been granted access
to by the host.

---

## How a build works (pipeline)

1. **Validate** — confirm there's an entry HTML file, CSS, JS, and detect SPA routing requirements.
2. **Prepare** — create a throwaway Go module (`%TEMP%\w2e\build-XXXX\host`).
3. **Embed** — copy the user's web assets into `host/web/`, generate a single self-contained `main.go` that:
   - serves them from an embedded `//go:embed all:web` FS,
   - listens on `127.0.0.1:0` (OS-assigned port, never `0.0.0.0`),
   - launches a WebView2 window pointed at the local server,
   - falls back to the entry HTML for SPA navigational routes,
   - routes `window.open` and external links to the system default browser.
4. **Compile** — `go mod tidy && go build -ldflags "-H windowsgui"` → produces `app.exe` (no console window).
5. **Verify** — open the EXE in binary mode and parse the PE header to confirm the subsystem is `IMAGE_SUBSYSTEM_WINDOWS_GUI` (2).
6. **Icon** — if an icon was provided, apply it via rcedit when available; otherwise record the chosen path in `w2e-icon.json` beside the EXE.
7. **Output** — copy the verified EXE to the requested output path.

The same `builder.Engine` powers CLI, MCP, and GUI — there is exactly one build pipeline in the codebase.

---

## Architecture

```text
┌─────────────────────────────────────────────────────────┐
│                         w2e.exe                           │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │   CLI      │  │   GUI      │  │   MCP (stdio)        │  │
│  │ (internal/ │  │ (internal/ │  │ (mcp/server.go)      │  │
│  │    cli/    │  │   ui/)     │  │   5 tools:           │  │
│  │  cmd_*.go) │  │ Liquid-    │  │  w2e_validate,       │  │
│  │            │  │ Glass UI   │  │  w2e_inspect,        │  │
│  │            │  │ embedded   │  │  w2e_build,          │  │
│  │            │  │ web assets │  │  w2e_doctor,          │  │
│  │            │  └─────┬──────┘  │  w2e_version          │  │
│  └─────┬──────┘             │      └──────────┬──────────┘  │
│        │                    │                 │             │
│        └──────────────┬─────┴─────────────────┘             │
│                       ▼                                     │
│              internal/builder.Engine                       │
│   (Validate → Prepare → Embed → Compile → Verify → Output) │
│                       │                                     │
│                       ▼                                     │
│    generates a temp Go module containing                    │
│    main.go (host runtime) + embedded web assets            │
│                       │                                     │
│                       ▼                                     │
│                  go build -H windowsgui                     │
│                       │                                     │
│                       ▼                                     │
│                The user's app.exe                            │
│         (standalone, embeds the web project,                │
│          launches WebView2 on a 127.0.0.1:0 server)        │
└─────────────────────────────────────────────────────────────┘
```

---

## Constraints honored

- ✅ Pure Go, `CGO_ENABLED=0` (no GCC / MinGW / Visual Studio)
- ✅ No Administrator / UAC elevation required
- ✅ No Node.js / Python on the user's machine
- ✅ No console window (`-H windowsgui`)
- ✅ No fixed localhost port — `net.Listen("tcp", "127.0.0.1:0")`
- ✅ Only `127.0.0.1`, never `0.0.0.0`
- ✅ WebView2 Runtime detection + fallback messaging
- ✅ Full i18n (zh-TW, zh-CN, en, ja, ko) with system-language detection
- ✅ One BuildEngine shared by GUI / CLI / MCP
- ✅ All GUI assets embedded (no internet dependency at launch)
- ✅ SPA route fallback to `index.html`
- ✅ `%LOCALAPPDATA%` for app data, never `Program Files`
- ✅ Stable machine-readable error codes
- ✅ PE verification on every output EXE
- ✅ Apple / iOS 26 "Liquid Glass" UI (translucent glassy blur, not Material / Bootstrap / cyberpunk), with light/dark/auto theme
- ✅ Accessibility: keyboard navigation, visible focus rings, ARIA roles, reduced-motion support, dual-pane SMS, `visually-hidden` labels

---

## Project layout

```text
cmd/
  w2e/             ← CLI + GUI entry point
  w2e-mcp/         ← standalone MCP server binary
internal/
  builder/         ← shared BuildEngine + host-runtime template + PE verify
  cli/             ← subcommand dispatch (build/validate/doctor/mcp/gui)
  config/          ← BuildConfig + BuildForm
  errcode/         ← stable error codes
  i18n/            ← embedded JSON locales (en, zh-TW, zh-CN, ja, ko)
  inspector/       ← project analysis
  logging/         ← leveled logger → %LOCALAPPDATA%\w2e\logs\
  runtime/         ← WebView2 Runtime registry detection
  ui/              ← Liquid Glass web UI (embedded assets)
  validator/       ← static project validation
  webserver/       ← MIME table + static file + SPA fallback server
  webview/         ← WebView2 window launcher (shared)
  icon/            ← icon conversion + rcedit application
mcp/               ← MCP server + tools + path-safety guards
tests/
  samples/basic-web/ ← sample project used by end-to-end smoke
```

---

## Testing

```powershell
go test ./...
```

The unit test suite covers the builder's `sanitizeAppID` and PE verifier,
the i18n bundle (locale switch + available-list + T() round-trip), the
validator (happy path, empty dir, `node_modules` skip), and the MCP server's
`rejectSystemPaths` + `newErrEnvelope` suggestion logic.

For an end-to-end smoke build:

```powershell
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\Smoke.exe
```

---

## License

MIT — see [LICENSE](LICENSE).
