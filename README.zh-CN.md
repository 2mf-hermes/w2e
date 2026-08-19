<p align="center">
  <b>🌐 请选择您的语言：</b><br>
  <a href="README.md">English</a> · <a href="README.zh-TW.md">繁體中文</a> · <a href="README.zh-CN.md">简体中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a>
</p>

# w2e — Web → 原生 EXE 打包工具（Windows · Linux · macOS）

w2e 能将任何 HTML/CSS/JS/SPA 项目打包为 Windows、Linux、macOS 上的**独立原生可执行文件**——一个 Web 项目，三种目标平台。每个生成的执行文件会启动系统内置的 Webview（Edge WebView2 / WebKit2GTK / WKWebView），连接到内嵌的 `127.0.0.1:0` 本机服务器，无需控制台窗口、无需管理员权限/UAC，用户的电脑也无需安装 Node.js / Python。


![w2e Screenshot](screenshot/screenshot.png)

一个代码库提供两款产品：

| 产品 | 功能说明 |
| --- | --- |
| **w2e Desktop**（`w2e.exe` / `w2e` / `w2e.app`） | Apple / iOS 26「Liquid Glass」玻璃效果 GUI。选择项目、调整窗口设置，点击**打包**即可生成执行文件。 |
| **w2e MCP Server**（`w2e mcp` 或 `w2e-mcp.exe`） | MCP stdio 服务器，提供 5 个工具，让 AI Agent（Claude Code、Codex、Gemini CLI…）能以编程方式验证 / 分析 / 打包 / 环境诊断 / 版本查询。 |

---

## 快速入门（Windows 主机）

```powershell
# 编译 w2e（需要 Go 1.23+，会自动下载 1.25 工具链）
# 使用 build.ps1 以确保 w2e.exe 链接为 GUI 子系统（无 DOS 窗口）：
.\build.ps1
.\build.ps1 -AlsoMcp          # 同时编译 bin\w2e-mcp.exe

# 将 Web 项目打包为 Windows EXE（现代机器约 6 秒）
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp.exe --title "My App"

# 一次跨编译三个平台：
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp --target all

# 启动 GUI：
.\bin\w2e.exe
```

输出结果：

| 目标平台 | 文件 | 格式 |
| --- | --- | --- |
| `windows` | `MyApp.exe`（`--target all` 时为 `MyApp-windows.exe`）| PE、GUI 子系统、无控制台窗口 |
| `linux` | `MyApp`（或 `MyApp-linux`）| ELF 64-bit |
| `darwin` | `MyApp`（或 `MyApp-darwin`）| Mach-O 64-bit |

所有平台上，最终用户只需安装**对应平台的 Webview 运行时**——Windows 上的 Edge WebView2 Runtime（Windows 11 已内置）、Linux 上的 WebKit2GTK（大部分桌面发行版已内置）、macOS 上的 WKWebView（已内置）。运行时无需 Go 工具链。

---

## 系统需求

| 角色 | 需求 |
| --- | --- |
| **Windows 编译主机** | Windows 10/11 x64/ARM64、**Go 1.23+**。跨编译到 Linux/macOS 需额外安装对应的 C 交叉编译器。 |
| **Linux 编译主机** | 任意现代 Linux、**Go 1.23+**、`gcc`、`libwebkit2gtk-4.1-dev`（+ `libgtk-3-dev`）。仅支持原生 Linux 编译。 |
| **macOS 编译主机** | macOS 10.13+、**Go 1.23+**、Xcode 命令行工具（`clang`）。仅支持原生 macOS 编译。 |
| **最终用户** | 对应平台的 Webview 运行时（WebView2 / WebKit2GTK / WKWebView）。 |

使用 `w2e doctor` 检查环境：

```text
w2e doctor — 环境诊断

  Windows x64:         ✓
  Go 工具链:           ✓
  WebView2 Runtime:    ✕（未安装）  <-- 仅在运行时需要
  临时目录:            ✓ C:\Users\you\AppData\Local\Temp
  用户数据文件夹:      ✓

编译目标:
  windows:              ✓ 原生（Go=true / C=true）
  linux:                ✕ 交叉编译（Go=true / C=false）
    缺少交叉 C 编译器：请安装对应工具链（见 README）
  darwin:               ✕ 交叉编译（Go=true / C=false）
    缺少交叉 C 编译器：请安装对应工具链（见 README）

build_available: true
cross_compile_available: false
```

> **交叉编译。** Windows→Windows 为纯 Go（`CGO_ENABLED=0`），只需 Go 工具链。编译 Linux/macOS 可执行文件需要 `CGO_ENABLED=1` 加上目标平台的 C 编译器。在原生主机上，Linux 使用 `gcc`、macOS 使用 `clang`（通过 `xcode-select --install` 安装）。从 Windows 交叉编译需安装：
>
> | 目标 | C 编译器 | 工具包 |
> | --- | --- | --- |
> | 从 Windows/macOS 交叉编译 linux/amd64 | `x86_64-linux-gnu-gcc` | `mingw-w64` / Zig ccache / Docker |
> | 从 Linux/Windows 交叉编译 darwin/amd64 | `o64-clang` | `osxcross`（需要 macOS SDK） |
>
> 最简便的方式是在各原生主机上执行 `w2e build --target all`（CI 矩阵）。Windows 上的单 EXE 部署维持纯 Go。

---

## CLI 参考

```text
w2e                      启动 GUI
w2e build SOURCE [flags] 将 Web 项目编译为原生可执行文件
  --entry FILE             指定入口 HTML 文件（默认：自动检测）
  --name NAME              应用程序名称（默认：输出文件名）
  --title TITLE            窗口标题（默认："My App"）
  --width N                初始窗口宽度（默认：1024，最小：320）
  --height N               初始窗口高度（默认：720，最小：240）
  --icon PATH              图标路径（.ico 或 .png）
  --no-resizable           禁止调整窗口大小
  --output PATH            输出路径（必填；--target all 时自动加后缀）
  --keep-temp              失败时保留临时目录
  --url URL                在线 URL 模式：执行文件直接加载指定网址
  --target PLATFORM        windows | linux | darwin | all（默认：windows）
w2e validate SOURCE     打包前验证 Web 项目
w2e doctor              诊断本机编译环境（含交叉编译目标）
w2e mcp                 启动 MCP 服务器（stdio 传输）
w2e version             显示版本信息
w2e help                显示帮助
```

### 在线 URL 模式

`w2e build --url https://...` 生成的执行文件会直接在 Webview 中加载指定网址——适合将已部署的 Web 应用程序打包为桌面启动器。

---

## MCP 集成

`w2e mcp`（或独立的 `w2e-mcp.exe`）通过 stdio 说 MCP（Model Context Protocol）。可接入任何支持 MCP stdio 的 Agent。配置示例：

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
| `w2e_validate` | 打包前验证 Web 项目 |
| `w2e_inspect` | 分析目录结构（框架、入口、文件数、SPA、警告） |
| `w2e_build` | 将本地 Web 项目或 `source_url` 打包为执行文件 |
| `w2e_doctor` | 回报主机编译环境状态 |
| `w2e_version` | 回报 w2e 版本信息 |

所有响应均为 JSON 格式。失败时包含 `error_code` 与机器可读的 `message`，必要时附带 `suggestion`。

### 安全性

`w2e_build` **拒绝** Windows 系统目录下的输出路径（`C:\Windows`、`C:\Program Files`、`C:\Program Files (x86)`、`C:\ProgramData`、`C:\Windows\System32`），并拒绝输出路径中的 `..` 相对路径穿越。源目录受限于 Agent 被授权访问的路径。

---

## 编译流程

1. **验证** — 确认存在入口 HTML 文件、CSS、JS，并检测 SPA 路由需求。
2. **准备** — 创建一次性 Go 模块（`%TEMP%\w2e\build-XXXX\host`）。
3. **嵌入** — 将 Web 资源复制到 `host/web/`，生成一个独立的 `main.go`：
   - 使用 `//go:embed all:web` 嵌入式文件系统提供服务
   - 监听 `127.0.0.1:0`（OS 随机分配，永不使用 `0.0.0.0`）
   - 启动 WebView2 窗口连接到本机服务器
   - SPA 导览路由回退至入口 HTML
   - `window.open` 与外部链接打开系统默认浏览器
4. **编译** — `go mod tidy && go build -ldflags "-H windowsgui"` → 生成 `app.exe`（无控制台窗口）
5. **验证** — 以二进制模式打开 EXE，解析 PE 文件头确认子系统为 `IMAGE_SUBSYSTEM_WINDOWS_GUI`（2）
6. **图标** — 若提供图标则通过 rcedit 套用；否则在 EXE 旁的 `w2e-icon.json` 记录路径
7. **输出** — 将验证过的 EXE 复制到指定输出路径

CLI、MCP 和 GUI 共用同一个 `builder.Engine`——代码库中只有一条编译管线。

---

## 架构

```text
┌─────────────────────────────────────────────────────────┐
│                         w2e.exe                           │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │   CLI      │  │   GUI      │  │   MCP (stdio)        │  │
│  │ (internal/ │  │ (internal/ │  │ (mcp/server.go)      │  │
│  │    cli/    │  │   ui/)     │  │   5 tools:           │  │
│  │  cmd_*.go) │  │ Liquid-    │  │  w2e_validate,       │  │
│  │            │  │ Glass UI   │  │  w2e_inspect,        │  │
│  │            │  │ 内嵌       │  │  w2e_build,          │  │
│  │            │  │ Web 资源   │  │  w2e_doctor,          │  │
│  │            │  └─────┬──────┘  │  w2e_version          │  │
│  └─────┬──────┘        │         └──────────┬──────────┘  │
│        │               │                    │             │
│        └───────────┬───┴────────────────────┘             │
│                    ▼                                      │
│           internal/builder.Engine                       │
│   (验证 → 准备 → 嵌入 → 编译 → 验证 → 输出)               │
│                    │                                      │
│                    ▼                                      │
│         生成临时 Go 模块（main.go + 嵌入 Web 资源）         │
│                    │                                      │
│                    ▼                                      │
│               go build -H windowsgui                     │
│                    │                                      │
│                    ▼                                      │
│             用户的 app.exe                                │
│      （独立执行，嵌入 Web 项目，                             │
│       在 127.0.0.1:0 启动 WebView2）                     │
└─────────────────────────────────────────────────────────────┘
```

---

## 设计原则

- ✅ 纯 Go，`CGO_ENABLED=0`（不需要 GCC / MinGW / Visual Studio）
- ✅ 不需要管理员 / UAC 权限
- ✅ 用户电脑不需要 Node.js / Python
- ✅ 无控制台窗口（`-H windowsgui`）
- ✅ 不使用固定端口 — `net.Listen("tcp", "127.0.0.1:0")`
- ✅ 仅绑定 `127.0.0.1`，永不使用 `0.0.0.0`
- ✅ WebView2 Runtime 检测与回退消息
- ✅ 完整多语言支持（zh-TW、zh-CN、en、ja、ko）+ 系统语言自动检测
- ✅ CLI / GUI / MCP 共用同一个 BuildEngine
- ✅ 所有 GUI 资源内嵌（启动时无需网络）
- ✅ SPA 路由回退至 `index.html`
- ✅ 使用 `%LOCALAPPDATA%` 存储应用程序数据
- ✅ 稳定的机器可读错误码
- ✅ 每个输出 EXE 均进行 PE 验证
- ✅ Apple / iOS 26「Liquid Glass」UI（半透明玻璃模糊效果），支持明亮 / 暗黑 / 自动主题
- ✅ 无障碍：键盘导航、可见焦点环、ARIA 角色、减少动态效果支持

---

## 项目结构

```text
cmd/
  w2e/             ← CLI + GUI 入口
  w2e-mcp/         ← 独立 MCP 服务器
internal/
  builder/         ← 共用 BuildEngine + 宿主运行时模板 + PE 验证
  cli/             ← 子命令分派（build/validate/doctor/mcp/gui）
  config/          ← BuildConfig + BuildForm
  errcode/         ← 稳定错误码
  i18n/            ← 嵌入式 JSON 语言包（en、zh-TW、zh-CN、ja、ko）
  inspector/       ← 项目分析
  logging/         ← 分级日志 → %LOCALAPPDATA%\w2e\logs\
  runtime/         ← WebView2 Runtime 注册表检测
  ui/              ← Liquid Glass Web UI（嵌入式资源）
  validator/       ← 静态项目验证
  webserver/       ← MIME 表 + 静态文件 + SPA 回退服务器
  webview/         ← WebView2 窗口启动器
  icon/            ← 图标转换 + rcedit 套用
mcp/               ← MCP 服务器 + 工具 + 路径安全防护
tests/
  samples/basic-web/ ← 端到端测试用示例项目
```

---

## 测试

```powershell
go test ./...
```

单元测试覆盖 `sanitizeAppID` 与 PE 验证器、i18n 语言包切换与 T() 往返测试、验证器（正常路径、空目录、跳过 `node_modules`）、以及 MCP 服务器的路径安全防护逻辑。

端到端冒烟测试：

```powershell
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\Smoke.exe
```

---

## 许可证

MIT — 详见 [LICENSE](LICENSE)。
