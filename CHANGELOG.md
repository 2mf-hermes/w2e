# Changelog

## v1.0.0 (2026-08-16)

### Initial Release

#### Features

- **w2e Desktop** — Apple / iOS 26 "Liquid Glass" GUI for packaging Web projects
- **w2e CLI** — `w2e build`, `w2e validate`, `w2e doctor`, `w2e mcp`, `w2e version`
- **w2e MCP Server** — 5 tools for AI Agent integration (`w2e_validate`, `w2e_inspect`, `w2e_build`, `w2e_doctor`, `w2e_version`)
- Cross-platform builds: Windows, Linux, macOS
- Embedded HTTP server with SPA fallback routing
- WebView2 Runtime detection and graceful fallback
- Complete i18n: English, 繁體中文, 简体中文, 日本語, 한국어 + auto-detect system language
- CLI / GUI / MCP share the same `builder.Engine`
- PE binary verification for all output executables
- macOS code signing support
- Icon embedding via rcedit
- Auto-updater with GitHub Releases integration
- Directory browsing with native OS dialogs
- Dark / Light / System theme support
- Accessibility: keyboard navigation, focus rings, ARIA roles, reduced motion

#### Platforms

| Platform | Binary | Runtime |
|----------|--------|---------|
| Windows x64 | `w2e-windows.exe` | Edge WebView2 (built-in on Windows 11) |
| Linux x64 | `w2e-linux` | WebKit2GTK (pre-installed on most desktop distros) |
| macOS ARM64 | `w2e-darwin-arm64` | WKWebView (built-in) |
| macOS x64 | `w2e-darwin-amd64` | WKWebView (built-in) |

#### Security

- Binds only to `127.0.0.1` (never `0.0.0.0`)
- Rejects output paths in Windows system directories
- Path traversal protection with `..` rejection
- Source directory restricted to granted access paths

#### Known Limitations

- Linux/macOS builds require CGO (native C compiler on build host)
- Windows builds are pure Go (CGO_ENABLED=0, no external toolchain)
