<p align="center">
  <b>🌐 선호하는 언어로 읽어주세요:</b><br>
  <a href="README.md">English</a> · <a href="README.zh-TW.md">繁體中文</a> · <a href="README.zh-CN.md">简体中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.ko.md">한국어</a>
</p>

# w2e — Web → 네이티브 EXE 패키저 (Windows · Linux · macOS)

w2e는 모든 HTML/CSS/JS/SPA Web 프로젝트를 Windows, Linux, macOS에서 **독립 실행형 네이티브 실행 파일**로 변환합니다 — 하나의 Web 프로젝트로 세 가지 대상 플랫폼을 지원합니다. 생성된 각 실행 파일은 시스템 내장 Webview(Edge WebView2 / WebKit2GTK / WKWebView)를 시작하여 내장된 `127.0.0.1:0` 로컬 서버에 연결합니다. 콘솔 창, 관리자 권한/UAC, Node.js/Python 런타임이 필요하지 않습니다.


![w2e Screenshot](screenshot.png)

하나의 코드베이스에서 두 가지 제품을 제공합니다:

| 제품 | 기능 |
| --- | --- |
| **w2e Desktop** (`w2e.exe` / `w2e` / `w2e.app`) | Apple / iOS 26 "Liquid Glass" GUI. 프로젝트를 선택하고 창 설정을 조정한 후 **패키징**을 클릭하면 실행 파일이 생성됩니다. |
| **w2e MCP Server** (`w2e mcp` 또는 `w2e-mcp.exe`) | MCP stdio 서버로 5개의 도구를 제공하여 AI Agent(Claude Code, Codex, Gemini CLI 등)가 프로그래밍 방식으로 검증 / 분석 / 빌드 / 환경 진단 / 버전 조회를 수행할 수 있습니다. |

---

## 빠른 시작 (Windows 호스트)

```powershell
# w2e 빌드 (Go 1.23+ 필요, 1.25 툴체인 자동 다운로드)
# build.ps1을 사용하여 w2e.exe를 GUI 서브시스템으로 링크 (DOS 창 없음):
.\build.ps1
.\build.ps1 -AlsoMcp          # bin\w2e-mcp.exe도 함께 빌드

# Web 프로젝트를 Windows EXE로 패키징 (최신 머신에서 약 6초)
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp.exe --title "My App"

# 세 플랫폼을 한 번에 크로스 컴파일:
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\MyApp --target all

# GUI 시작:
.\bin\w2e.exe
```

출력 결과:

| 대상 | 파일 | 형식 |
| --- | --- | --- |
| `windows` | `MyApp.exe` (`--target all`일 때 `MyApp-windows.exe`) | PE, GUI 서브시스템, 콘솔 창 없음 |
| `linux` | `MyApp` (또는 `MyApp-linux`) | ELF 64-bit |
| `darwin` | `MyApp` (또는 `MyApp-darwin`) | Mach-O 64-bit |

모든 플랫폼에서 최종 사용자는 **해당 플랫폼의 Webview 런타임**만 설치하면 됩니다 — Windows의 Edge WebView2 Runtime(Windows 11에 포함), Linux의 WebKit2GTK(대부분의 데스크톱 배포판에 포함), macOS의 WKWebView(내장). 런타임에 Go 툴체인은 불필요합니다.

---

## 시스템 요구사항

| 역할 | 요구사항 |
| --- | --- |
| **Windows 빌드 호스트** | Windows 10/11 x64/ARM64, **Go 1.23+**. Linux/macOS 크로스 컴파일에는 대응하는 C 크로스 컴파일러가 필요합니다. |
| **Linux 빌드 호스트** | 최신 Linux, **Go 1.23+**, `gcc`, `libwebkit2gtk-4.1-dev` (+ `libgtk-3-dev`). 네이티브 Linux 빌드만 지원. |
| **macOS 빌드 호스트** | macOS 10.13+, **Go 1.23+**, Xcode 명령줄 도구(`clang`). 네이티브 macOS 빌드만 지원. |
| **최종 사용자** | 해당 플랫폼의 Webview 런타임 (WebView2 / WebKit2GTK / WKWebView). |

`w2e doctor`로 환경을 확인합니다:

```text
w2e doctor — 환경 진단

  Windows x64:         ✓
  Go 툴체인:           ✓
  WebView2 Runtime:    ✕ (미설치)   <-- 런타임에만 필요
  임시 디렉토리:        ✓ C:\Users\you\AppData\Local\Temp
  사용자 데이터 폴더:   ✓

빌드 대상:
  windows:              ✓ 네이티브 (Go=true / C=true)
  linux:                ✕ 크로스 컴파일 (Go=true / C=false)
    크로스 C 컴파일러 없음: 대응하는 툴체인을 설치하세요 (README 참조)
  darwin:               ✕ 크로스 컴파일 (Go=true / C=false)
    크로스 C 컴파일러 없음: 대응하는 툴체인을 설치하세요 (README 참조)

build_available: true
cross_compile_available: false
```

> **크로스 컴파일.** Windows→Windows는 순수 Go(`CGO_ENABLED=0`)로 Go 툴체인만 필요합니다. Linux/macOS 바이너리 빌드에는 `CGO_ENABLED=1`과 대상용 C 컴파일러가 필요합니다. 네이티브 호스트에서는 Linux는 `gcc`, macOS는 `clang`(`xcode-select --install`로 설치)을 사용합니다. Windows에서 크로스 컴파일:
>
> | 대상 | C 컴파일러 | 키트 |
> | --- | --- | --- |
> | Windows/macOS에서 linux/amd64 | `x86_64-linux-gnu-gcc` | `mingw-w64` / Zig ccache / Docker |
> | Linux/Windows에서 darwin/amd64 | `o64-clang` | `osxcross` (macOS SDK 필요) |
>
> 가장 쉬운 방법은 각 네이티브 호스트에서 `w2e build --target all`을 실행하는 것입니다 (CI 매트릭스). Windows의 단일 EXE 배포는 순수 Go를 유지합니다.

---

## CLI 레퍼런스

```text
w2e                      GUI 시작
w2e build SOURCE [flags] Web 프로젝트에서 네이티브 실행 파일 빌드
  --entry FILE             진입 HTML 파일 오버라이드 (기본: 자동 감지)
  --name NAME              애플리케이션 이름 (기본: 출력 기본 이름)
  --title TITLE            창 제목 (기본: "My App")
  --width N                초기 창 너비 (기본: 1024, 최소: 320)
  --height N               초기 창 높이 (기본: 720, 최소: 240)
  --icon PATH              아이콘 경로 (.ico 또는 .png)
  --no-resizable           창 리사이징 비활성화
  --output PATH            출력 경로 (필수; --target all일 때 접미사 추가)
  --keep-temp              실패 시 임시 디렉토리 유지
  --url URL                온라인 URL 모드: 바이너리가 이 URL을 직접 로드
  --target PLATFORM        windows | linux | darwin | all (기본: windows)
w2e validate SOURCE     빌드 전 Web 프로젝트 검증
w2e doctor              로컬 빌드 환경 진단 (크로스 대상 포함)
w2e mcp                 MCP 서버 시작 (stdio 전송)
w2e version             버전 정보 출력
w2e help                도움말 표시
```

### 온라인 URL 모드

`w2e build --url https://...`는 지정된 URL을 플랫폼 Webview에서 직접 로드하는 바이너리를 생성합니다 — 호스팅된 앱을 데스크톱 런처로 패키징하는 데 유용합니다.

---

## MCP 통합

`w2e mcp`(또는 전용 `w2e-mcp.exe` 바이너리)는 stdio를 통해 MCP(Model Context Protocol)를 사용합니다. MCP stdio 서버를 지원하는 모든 Agent에 연결할 수 있습니다. 설정 예:

```jsonc
{
  "mcpServers": {
    "w2e": {
      "command": "C:\\path\\to\\bin\\w2e-mcp.exe"
    }
  }
}
```

### 제공 도구

| 도구 | 용도 |
| --- | --- |
| `w2e_validate` | 패키징 전 Web 프로젝트 검증 |
| `w2e_inspect` | 디렉토리 분석 (프레임워크, 진입점, 파일 수, SPA, 경고) |
| `w2e_build` | 로컬 Web 프로젝트 또는 `source_url`을 EXE로 빌드 |
| `w2e_doctor` | 호스트 빌드 환경 상태 보고 |
| `w2e_version` | w2e 버전 정보 보고 |

모든 응답은 JSON 형식입니다. 실패 시 `error_code`와 기계 판독 가능한 `message`, 필요 시 `suggestion`이 포함됩니다.

### 보안

`w2e_build`는 Windows 시스템 디렉토리 내 출력 경로를 **거부**합니다 (`C:\Windows`, `C:\Program Files`, `C:\Program Files (x86)`, `C:\ProgramData`, `C:\Windows\System32`) 및 출력 경로의 상대 `..` 트래버сал도 거부합니다. 소스 디렉토리는 호스트가 Agent에 부여한 접근 권한이 있는 경로로 제한됩니다.

---

## 빌드 파이프라인

1. **검증** — 진입 HTML 파일, CSS, JS 존재 확인 및 SPA 라우팅 요구사항 감지.
2. **준비** — 일회용 Go 모듈 생성 (`%TEMP%\w2e\build-XXXX\host`).
3. **임베드** — Web 자산을 `host/web/`에 복사하고 독립형 `main.go` 생성:
   - `//go:embed all:web` 임베디드 파일시스템으로 제공
   - `127.0.0.1:0`에서 리슨 (OS 할당, `0.0.0.0` 사용 안 함)
   - 로컬 서버를 가리키는 WebView2 창 시작
   - SPA 내비게이션 경로는 진입 HTML로 폴백
   - `window.open` 및 외부 링크는 시스템 기본 브라우저에서 열기
4. **컴파일** — `go mod tidy && go build -ldflags "-H windowsgui"` → `app.exe` 생성 (콘솔 창 없음)
5. **검증** — 바이너리 모드로 EXE를 열고 PE 헤더를 파싱하여 서브시스템이 `IMAGE_SUBSYSTEM_WINDOWS_GUI`(2)인지 확인
6. **아이콘** — 아이콘이 제공되면 rcedit으로 적용; 그렇지 않으면 `w2e-icon.json`에 경로 기록
7. **출력** — 검증된 EXE를 지정된 출력 경로로 복사

CLI, MCP, GUI는 동일한 `builder.Engine`을 공유합니다 — 코드베이스에 빌드 파이프라인은 정확히 하나입니다.

---

## 아키텍처

```text
┌─────────────────────────────────────────────────────────┐
│                         w2e.exe                           │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐  │
│  │   CLI      │  │   GUI      │  │   MCP (stdio)        │  │
│  │ (internal/ │  │ (internal/ │  │ (mcp/server.go)      │  │
│  │    cli/    │  │   ui/)     │  │   5개 도구:          │  │
│  │  cmd_*.go) │  │ Liquid-    │  │  w2e_validate,       │  │
│  │            │  │ Glass UI   │  │  w2e_inspect,        │  │
│  │            │  │ 내장       │  │  w2e_build,          │  │
│  │            │  │ Web 자산   │  │  w2e_doctor,          │  │
│  │            │  └─────┬──────┘  │  w2e_version          │  │
│  └─────┬──────┘        │         └──────────┬──────────┘  │
│        │               │                    │             │
│        └───────────┬───┴────────────────────┘             │
│                    ▼                                      │
│           internal/builder.Engine                       │
│   (검증 → 준비 → 임베드 → 컴파일 → 검증 → 출력)             │
│                    │                                      │
│                    ▼                                      │
│       임시 Go 모듈 생성 (main.go + 내장 Web 자산)           │
│                    │                                      │
│                    ▼                                      │
│               go build -H windowsgui                     │
│                    │                                      │
│                    ▼                                      │
│             사용자의 app.exe                               │
│      (독립 실행, Web 프로젝트 내장,                           │
│       127.0.0.1:0에서 WebView2 시작)                     │
└─────────────────────────────────────────────────────────────┘
```

---

## 설계 원칙

- ✅ 순수 Go, `CGO_ENABLED=0` (GCC / MinGW / Visual Studio 불필요)
- ✅ 관리자 / UAC 권한 불필요
- ✅ 사용자 머신에 Node.js / Python 불필요
- ✅ 콘솔 창 없음 (`-H windowsgui`)
- ✅ 고정 포트 미사용 — `net.Listen("tcp", "127.0.0.1:0")`
- ✅ `127.0.0.1`만 바인딩, `0.0.0.0` 사용 안 함
- ✅ WebView2 Runtime 감지 + 폴백 메시지
- ✅ 완전한 다국어 지원 (zh-TW, zh-CN, en, ja, ko) + 시스템 언어 자동 감지
- ✅ CLI / GUI / MCP가 동일한 BuildEngine 공유
- ✅ 모든 GUI 자산 내장 (시작 시 인터넷 불필요)
- ✅ SPA 경로를 `index.html`로 폴백
- ✅ `%LOCALAPPDATA%`에 애플리케이션 데이터 저장
- ✅ 안정적인 기계 판독 가능 오류 코드
- ✅ 모든 출력 EXE에 PE 검증
- ✅ Apple / iOS 26 "Liquid Glass" UI (반투명 유리 블러), 라이트 / 다크 / 자동 테마 지원
- ✅ 접근성: 키보드 탐색, 포커스 링, ARIA 역할, 모션 축소 지원

---

## 프로젝트 구조

```text
cmd/
  w2e/             ← CLI + GUI 진입점
  w2e-mcp/         ← 독립 MCP 서버
internal/
  builder/         ← 공유 BuildEngine + 호스트 런타임 템플릿 + PE 검증
  cli/             ← 서브커맨드 디스패치 (build/validate/doctor/mcp/gui)
  config/          ← BuildConfig + BuildForm
  errcode/         ← 안정적인 오류 코드
  i18n/            ← 내장 JSON 로케일 (en, zh-TW, zh-CN, ja, ko)
  inspector/       ← 프로젝트 분석
  logging/         ← 레벨별 로거 → %LOCALAPPDATA%\w2e\logs\
  runtime/         ← WebView2 Runtime 레지스트리 감지
  ui/              ← Liquid Glass Web UI (내장 자산)
  validator/       ← 정적 프로젝트 검증
  webserver/       ← MIME 테이블 + 정적 파일 + 폴백 서버
  webview/         ← WebView2 창 런처
  icon/            ← 아이콘 변환 + rcedit 적용
mcp/               ← MCP 서버 + 도구 + 경로 보안 가드
tests/
  samples/basic-web/ ← 엔드투엔드 테스트용 샘플 프로젝트
```

---

## 테스트

```powershell
go test ./...
```

단위 테스트는 `sanitizeAppID`와 PE 검증, i18n 로케일 전환과 T() 왕복 테스트, 검증기(정상 경로, 빈 디렉토리, `node_modules` 건너뛰기), MCP 서버의 경로 보안 가드를 다룹니다.

엔드투엔드 스모크 빌드:

```powershell
.\bin\w2e.exe build .\tests\samples\basic-web --output .\dist\Smoke.exe
```

---

## 라이선스

MIT — [LICENSE](LICENSE) 참조.
