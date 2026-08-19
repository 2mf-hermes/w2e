<p align="center">
  <b>🌐 請選擇您的語言：</b><br>
  <a href="README.md">English</a> · <a href="README.zh-TW.md">繁體中文</a> · <a href="README.zh-CN.md">简体中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a>
</p>

# w2e — Web → 原生 EXE 打包工具（Windows · Linux · macOS）

w2e 能將任何 HTML/CSS/JS/SPA 專案打包為 Windows、Linux、macOS 上的**獨立原生執行檔**——一個 Web 專案，三種目標平台。每個產生的執行檔會啟動系統內建的 Webview（Edge WebView2 / WebKit2GTK / WKWebView），連線到內嵌的 `127.0.0.1:0` 本機伺服器，不需要控制台視窗、不需要管理員權限/UAC，使用者的電腦也無需安裝 Node.js / Python。


![w2e Screenshot](screenshot.png)

一個程式碼庫提供兩項產品：

| 產品 | 功能說明 |
| --- | --- |
| **w2e Desktop**（`w2e.exe` / `w2e` / `w2e.app`） | Apple / iOS 26「Liquid Glass」玻璃效果 GUI。選擇專案、調整視窗設定，點擊**打包**即可產生執行檔。 |
| **w2e MCP Server**（`w2e mcp` 或 `w2e-mcp.exe`） | MCP stdio 伺服器，提供 5 個工具，讓 AI Agent（Claude Code、Codex、Gemini CLI…）能以程式化方式驗證 / 分析 / 打包 / 環境診斷 / 版本查詢。 |

---

## 快速入門（Windows 主機）

```powershell
# 編譯 w2e（需要 Go 1.23+，會自動下載 1.25 工具鏈）
# 使用 build.ps1 以確保 w2e.exe 連結為 GUI 子系統（無 DOS 視窗）：
.\build.ps1
.\build.ps1 -AlsoMcp          # 同時編譯 bin\w2e-mcp.exe

# 將 Web 專案打包為 Windows EXE（現代機器約 6 秒）
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp.exe --title "My App"

# 一次跨編譯三個平台：
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp --target all

# 啟動 GUI：
.\bin\w2e.exe
```

輸出結果：

| 目標平台 | 檔案 | 格式 |
| --- | --- | --- |
| `windows` | `MyApp.exe`（`--target all` 時為 `MyApp-windows.exe`）| PE、GUI 子系統、無控制台視窗 |
| `linux` | `MyApp`（或 `MyApp-linux`）| ELF 64-bit |
| `darwin` | `MyApp`（或 `MyApp-darwin`）| Mach-O 64-bit |

所有平台上，最終使用者只需安裝**對應平台的 Webview 執行時**——Windows 上的 Edge WebView2 Runtime（Windows 11 已內建）、Linux 上的 WebKit2GTK（大部分桌面發行版已內建）、macOS 上的 WKWebView（已內建）。執行時不需要 Go 工具鏈。

---

## 系統需求

| 角色 | 需求 |
| --- | --- |
| **Windows 編譯主機** | Windows 10/11 x64/ARM64、**Go 1.23+**。跨編譯到 Linux/macOS 需額外安裝對應的 C 交叉編譯器。 |
| **Linux 編譯主機** | 任意現代 Linux、**Go 1.23+**、`gcc`、`libwebkit2gtk-4.1-dev`（+ `libgtk-3-dev`）。僅支援原生 Linux 編譯。 |
| **macOS 編譯主機** | macOS 10.13+、**Go 1.23+**、Xcode 命令列工具（`clang`）。僅支援原生 macOS 編譯。 |
| **最終使用者** | 對應平台的 Webview 執行時（WebView2 / WebKit2GTK / WKWebView）。 |

使用 `w2e doctor` 檢查環境：

```text
w2e doctor — 環境診斷

  Windows x64:         ✓
  Go 工具鏈:           ✓
  WebView2 Runtime:    ✕（未安裝）  <-- 僅在執行時需要
  臨時目錄:            ✓ C:\Users\you\AppData\Local\Temp
  使用者資料資料夾:    ✓

編譯目標:
  windows:              ✓ 原生（Go=true / C=true）
  linux:                ✕ 交叉編譯（Go=true / C=false）
    缺少交叉 C 編譯器：請安裝對應工具鏈（見 README）
  darwin:               ✕ 交叉編譯（Go=true / C=false）
    缺少交叉 C 編譯器：請安裝對應工具鏈（見 README）

build_available: true
cross_compile_available: false
```

> **交叉編譯。** Windows→Windows 為純 Go（`CGO_ENABLED=0`），只需要 Go 工具鏈。編譯 Linux/macOS 執行檔需要 `CGO_ENABLED=1` 加上目標平台的 C 編譯器。在原生主機上，Linux 使用 `gcc`、macOS 使用 `clang`（透過 `xcode-select --install` 安裝）。從 Windows 交叉編譯需安裝：
>
> | 目標 | C 編譯器 | 工具包 |
> | --- | --- | --- |
> | 從 Windows/macOS 交叉編譯 linux/amd64 | `x86_64-linux-gnu-gcc` | `mingw-w64` / Zig ccache / Docker |
> | 從 Linux/Windows 交叉編譯 darwin/amd64 | `o64-clang` | `osxcross`（需要 macOS SDK） |
>
> 最簡便的方式是在各原生主機上執行 `w2e build --target all`（CI 矩陣）。Windows 上的單一 EXE 部署維持純 Go。

---

## CLI 參考

```text
w2e                      啟動 GUI
w2e build SOURCE [flags] 將 Web 專案編譯為原生執行檔
  --entry FILE             指定入口 HTML 檔案（預設：自動偵測）
  --name NAME              應用程式名稱（預設：輸出檔名）
  --title TITLE            視窗標題（預設："My App"）
  --width N                初始視窗寬度（預設：1024，最小：320）
  --height N               初始視窗高度（預設：720，最小：240）
  --icon PATH              圖示路徑（.ico 或 .png）
  --no-resizable           禁止調整視窗大小
  --output PATH            輸出路徑（必填；--target all 時自動加後綴）
  --keep-temp              失敗時保留臨時目錄
  --url URL                線上 URL 模式：執行檔直接載入指定網址
  --target PLATFORM        windows | linux | darwin | all（預設：windows）
w2e validate SOURCE     打包前驗證 Web 專案
w2e doctor              診斷本機編譯環境（含交叉編譯目標）
w2e mcp                 啟動 MCP 伺服器（stdio 傳輸）
w2e version             顯示版本資訊
w2e help                顯示說明
```

### 線上 URL 模式

`w2e build --url https://...` 產生的執行檔會直接在 Webview 中載入指定網址——適合將已部署的 Web 應用程式打包為桌面啟動器。

---

## MCP 整合

`w2e mcp`（或獨立的 `w2e-mcp.exe`）透過 stdio 說 MCP（Model Context Protocol）。可接入任何支援 MCP stdio 的 Agent。設定範例：

```jsonc
{
  "mcpServers": {
    "w2e": {
      "command": "C:\\path\\to\\bin\\w2e-mcp.exe"
    }
  }
}
```

### 提供的工具

| 工具 | 用途 |
| --- | --- |
| `w2e_validate` | 打包前驗證 Web 專案 |
| `w2e_inspect` | 分析目錄結構（框架、入口、檔案數、SPA、警告） |
| `w2e_build` | 將本地 Web 專案或 `source_url` 打包為執行檔 |
| `w2e_doctor` | 回報主機編譯環境狀態 |
| `w2e_version` | 回報 w2e 版本資訊 |

所有回應均為 JSON 格式。失敗時包含 `error_code` 與可機器讀取的 `message`，必要時附帶 `suggestion`。

### 安全性

`w2e_build` **拒絕** Windows 系統目錄下的輸出路徑（`C:\Windows`、`C:\Program Files`、`C:\Program Files (x86)`、`C:\ProgramData`、`C:\Windows\System32`），並拒絕輸出路徑中的 `..` 相對路徑穿越。來源目錄受限於 Agent 被授權存取的路徑。

---

## 編譯流程

1. **驗證** — 確認存在入口 HTML 檔案、CSS、JS，並偵測 SPA 路由需求。
2. **準備** — 建立一次性 Go 模組（`%TEMP%\w2e\build-XXXX\host`）。
3. **嵌入** — 將 Web 資源複製到 `host/web/`，產生一個獨立的 `main.go`：
   - 使用 `//go:embed all:web` 嵌入式檔案系統提供服務
   - 監聽 `127.0.0.1:0`（OS 隨機分配，永不使用 `0.0.0.0`）
   - 啟動 WebView2 視窗連線到本機伺服器
   - SPA 導覽路由回退至入口 HTML
   - `window.open` 與外部連結開啟系統預設瀏覽器
4. **編譯** — `go mod tidy && go build -ldflags "-H windowsgui"` → 產生 `app.exe`（無控制台視窗）
5. **驗證** — 以二進位模式開啟 EXE，解析 PE 檔頭確認子系統為 `IMAGE_SUBSYSTEM_WINDOWS_GUI`（2）
6. **圖示** — 若提供圖示則透過 rcedit 套用；否則在 EXE 旁的 `w2e-icon.json` 記錄路徑
7. **輸出** — 將驗證過的 EXE 複製到指定輸出路徑

CLI、MCP 和 GUI 共用同一個 `builder.Engine`——程式碼庫中只有一條編譯管線。

---

## 架構

```text
┌─────────────────────────────────────────────────────────┐
│                         w2e.exe                           │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │   CLI      │  │   GUI      │  │   MCP (stdio)        │  │
│  │ (internal/ │  │ (internal/ │  │ (mcp/server.go)      │  │
│  │    cli/    │  │   ui/)     │  │   5 tools:           │  │
│  │  cmd_*.go) │  │ Liquid-    │  │  w2e_validate,       │  │
│  │            │  │ Glass UI   │  │  w2e_inspect,        │  │
│  │            │  │ 嵌入式     │  │  w2e_build,          │  │
│  │            │  │ Web 資源   │  │  w2e_doctor,          │  │
│  │            │  └─────┬──────┘  │  w2e_version          │  │
│  └─────┬──────┘        │         └──────────┬──────────┘  │
│        │               │                    │             │
│        └───────────┬───┴────────────────────┘             │
│                    ▼                                      │
│           internal/builder.Engine                       │
│   (驗證 → 準備 → 嵌入 → 編譯 → 驗證 → 輸出)               │
│                    │                                      │
│                    ▼                                      │
│         產生臨時 Go 模組（main.go + 嵌入 Web 資源）         │
│                    │                                      │
│                    ▼                                      │
│               go build -H windowsgui                     │
│                    │                                      │
│                    ▼                                      │
│             使用者的 app.exe                               │
│      （獨立執行，嵌入 Web 專案，                             │
│       在 127.0.0.1:0 啟動 WebView2）                     │
└─────────────────────────────────────────────────────────────┘
```

---

## 設計原則

- ✅ 純 Go，`CGO_ENABLED=0`（不需要 GCC / MinGW / Visual Studio）
- ✅ 不需要管理員 / UAC 權限
- ✅ 使用者電腦不需要 Node.js / Python
- ✅ 無控制台視窗（`-H windowsgui`）
- ✅ 不使用固定連接埠 — `net.Listen("tcp", "127.0.0.1:0")`
- ✅ 僅綁定 `127.0.0.1`，永不使用 `0.0.0.0`
- ✅ WebView2 Runtime 偵測與回退訊息
- ✅ 完整多語言支援（zh-TW、zh-CN、en、ja、ko）+ 系統語言自動偵測
- ✅ CLI / GUI / MCP 共用同一個 BuildEngine
- ✅ 所有 GUI 資源嵌入（啟動時不需網路）
- ✅ SPA 路由回退至 `index.html`
- ✅ 使用 `%LOCALAPPDATA%` 儲存應用程式資料
- ✅ 穩定的機器可讀取錯誤碼
- ✅ 每個輸出 EXE 均進行 PE 驗證
- ✅ Apple / iOS 26「Liquid Glass」UI（半透明玻璃模糊效果），支援明亮 / 暗黑 / 自動主題
- ✅ 無障礙：鍵盤導覽、可見焦點環、ARIA 角色、減少動態效果支援

---

## 專案結構

```text
cmd/
  w2e/             ← CLI + GUI 入口
  w2e-mcp/         ← 獨立 MCP 伺服器
internal/
  builder/         ← 共用 BuildEngine + 宿主執行時模板 + PE 驗證
  cli/             ← 子命令分派（build/validate/doctor/mcp/gui）
  config/          ← BuildConfig + BuildForm
  errcode/         ← 穩定錯誤碼
  i18n/            ← 嵌入式 JSON 語言包（en、zh-TW、zh-CN、ja、ko）
  inspector/       ← 專案分析
  logging/         ← 分級日誌 → %LOCALAPPDATA%\w2e\logs\
  runtime/         ← WebView2 Runtime 註冊表偵測
  ui/              ← Liquid Glass Web UI（嵌入式資源）
  validator/       ← 靜態專案驗證
  webserver/       ← MIME 表 + 靜態檔案 + SPA 回退伺服器
  webview/         ← WebView2 視窗啟動器
  icon/            ← 圖示轉換 + rcedit 套用
mcp/               ← MCP 伺服器 + 工具 + 路徑安全防護
tests/
  samples/basic-web/ ← 端對端測試用範例專案
```

---

## 測試

```powershell
go test ./...
```

單元測試涵蓋 `sanitizeAppID` 與 PE 驗證器、i18n 語言包切換與 T() 往返測試、驗證器（正常路徑、空目錄、跳過 `node_modules`）、以及 MCP 伺服器的路徑安全防護邏輯。

端對端冒煙測試：

```powershell
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\Smoke.exe
```

---

## 授權條款

MIT — 詳見 [LICENSE](LICENSE)。
