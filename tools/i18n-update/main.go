// Batch-add target-platform and UI strings to every w2e locale JSON, keyed in
// alphabetical order so diffs stay readable and JSON stays canonical.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// perLocale holds the new/updated strings keyed by locale code. Any locale not
// listed here gets the en default so nothing goes missing during lookup.
var perLocale = map[string]map[string]string{
	"en": {
		"app.tagline":              "Web → Native App",
		"build.subtitle":           "Pick a project, choose target platforms, and produce a native executable.",
		"build.title":              "Package a Web App",
		"field.output":             "Output path",
		"field.target":             "Target platform",
		"field.target.hint":        "Windows is pure Go. Linux & macOS need a C toolchain on the host.",
		"target.all":               "All platforms",
		"target.all.desc":          "Windows, Linux, and macOS in one build",
		"target.darwin":            "macOS",
		"target.darwin.desc":       "Mach-O executable (WKWebView)",
		"target.linux":             "Linux",
		"target.linux.desc":        "ELF executable (WebKit2GTK)",
		"target.windows":           "Windows",
		"target.windows.desc":       "EXE, GUI subsystem (WebView2)",
		"status.notReady":          "Validation produced warnings or errors — see below.",
		"status.ready":             "Project is ready for build.",
		"status.building":          "Packaging…",
		"status.done":              "Done",
		"status.needSource":        "Provide a source directory first.",
		"status.needOutput":        "Specify an output path.",
		"status.failed":            "Build failed",
		"status.targetPartial":    "Some targets failed — see details below.",
		"status.targetAllDone":     "All requested binaries produced.",
		"status.buildStarting":     "Starting build…",
		"hint.sourceDir":          "Must contain an index.html or equivalent entry file.",
		"hint.browse":              "Type or paste your project directory into the field.",
		"hint.browseIcon":          "Type or paste an .ico or .png icon path.",
		"action.browse":            "Browse…",
		"action.start":             "開始打包",
		"action.validate":          "Validate",
		"footer.note":              "Standalone native app generator — Windows · Linux · macOS",
	},
	"zh-TW": {
		"app.tagline":              "Web → 原生應用",
		"build.subtitle":           "選擇專案、挑選目標平台,產出可直接執行的原生檔案。",
		"build.title":              "打包網頁應用",
		"field.output":             "輸出路徑",
		"field.target":             "目標平台",
		"field.target.hint":        "Windows 為純 Go;Linux 與 macOS 需主機有 C 工具鏈。",
		"target.all":               "全部平台",
		"target.all.desc":          "一次產出 Windows、Linux、macOS 三個檔案",
		"target.darwin":            "macOS",
		"target.darwin.desc":       "Mach-O 執行檔(WKWebView)",
		"target.linux":             "Linux",
		"target.linux.desc":        "ELF 執行檔(WebKit2GTK)",
		"target.windows":           "Windows",
		"target.windows.desc":      "EXE,GUI 子系統(WebView2)",
		"status.notReady":          "驗證出現警告或錯誤 — 見下方詳情。",
		"status.ready":             "專案已就緒,可開始打包。",
		"status.building":          "打包中…",
		"status.done":              "完成",
		"status.needSource":        "請先填入來源目錄。",
		"status.needOutput":        "請指定輸出路徑。",
		"status.failed":            "打包失敗",
		"status.targetPartial":     "部分目標失敗 — 見下方詳情。",
		"status.targetAllDone":     "所有要求的執行檔已產出。",
		"status.buildStarting":     "開始打包…",
		"hint.sourceDir":           "目錄需含 index.html 或同等入口檔。",
		"hint.browse":              "請直接在欄位輸入或貼上專案目錄路徑。",
		"hint.browseIcon":          "請輸入或貼上 .ico / .png 圖示路徑。",
		"action.browse":            "瀏覽…",
		"action.start":             "開始打包",
		"action.validate":          "驗證",
		"footer.note":              "獨立原生應用產生器 — Windows · Linux · macOS",
	},
	"zh-CN": {
		"app.tagline":              "Web → 原生应用",
		"build.subtitle":           "选择项目、挑选目标平台,生成可直接执行的原生文件。",
		"build.title":              "打包网页应用",
		"field.output":             "输出路径",
		"field.target":             "目标平台",
		"field.target.hint":        "Windows 为纯 Go;Linux 与 macOS 需主机有 C 工具链。",
		"target.all":               "全部平台",
		"target.all.desc":          "一次生成 Windows、Linux、macOS 三个文件",
		"target.darwin":            "macOS",
		"target.darwin.desc":       "Mach-O 可执行文件(WKWebView)",
		"target.linux":             "Linux",
		"target.linux.desc":        "ELF 可执行文件(WebKit2GTK)",
		"target.windows":           "Windows",
		"target.windows.desc":      "EXE,GUI 子系统(WebView2)",
		"status.notReady":          "验证出现警告或错误 — 见下方详情。",
		"status.ready":             "项目已就绪,可开始打包。",
		"status.building":          "打包中…",
		"status.done":              "完成",
		"status.needSource":        "请先填入来源目录。",
		"status.needOutput":        "请指定输出路径。",
		"status.failed":            "打包失败",
		"status.targetPartial":     "部分目标失败 — 见下方详情。",
		"status.targetAllDone":     "所有要求的可执行文件已生成。",
		"status.buildStarting":     "开始打包…",
		"hint.sourceDir":           "目录需含 index.html 或同等入口文件。",
		"hint.browse":              "请直接在字段输入或粘贴项目目录路径。",
		"hint.browseIcon":          "请输入或粘贴 .ico / .png 图标路径。",
		"action.browse":            "浏览…",
		"action.start":             "开始打包",
		"action.validate":          "验证",
		"footer.note":              "独立原生应用生成器 — Windows · Linux · macOS",
	},
	"ja": {
		"app.tagline":              "Web → ネイティブアプリ",
		"build.subtitle":           "プロジェクトを選び、ターゲットを選んでネイティブ実行ファイルを生成します。",
		"build.title":              "Web アプリをパッケージ化",
		"field.output":             "出力パス",
		"field.target":             "ターゲット プラットフォーム",
		"field.target.hint":        "Windows は純 Go。Linux・macOS はホストに C ツールチェーンが必要。",
		"target.all":               "全プラットフォーム",
		"target.all.desc":          "Windows・Linux・macOS を一括生成",
		"target.darwin":            "macOS",
		"target.darwin.desc":       "Mach-O 実行ファイル(WKWebView)",
		"target.linux":             "Linux",
		"target.linux.desc":        "ELF 実行ファイル(WebKit2GTK)",
		"target.windows":           "Windows",
		"target.windows.desc":      "EXE・GUI サブシステム(WebView2)",
		"status.notReady":          "検証で警告またはエラーが見つかりました — 以下を参照。",
		"status.ready":             "プロジェクトはビルド可能です。",
		"status.building":          "パッケージ化中…",
		"status.done":              "完了",
		"status.needSource":        "ソース ディレクトリを入力してください。",
		"status.needOutput":        "出力パスを指定してください。",
		"status.failed":            "ビルド失敗",
		"status.targetPartial":     "一部ターゲットが失敗 — 以下を参照。",
		"status.targetAllDone":     "要求した実行ファイルをすべて生成しました。",
		"status.buildStarting":     "ビルドを開始…",
		"hint.sourceDir":           "index.html または同等のエントリ ファイルが必要。",
		"hint.browse":              "欄にプロジェクト ディレクトリ パスを直接入力または貼り付け。",
		"hint.browseIcon":          ".ico / .png アイコン パスを入力または貼り付け。",
		"action.browse":            "参照…",
		"action.start":             "開始打包",
		"action.validate":          "検証",
		"footer.note":              "スタンドアロン ネイティブ アプリ生成 — Windows · Linux · macOS",
	},
	"ko": {
		"app.tagline":              "Web → 네이티브 앱",
		"build.subtitle":           "프로젝트를 선택하고 타겟 플랫폼을 골라 네이티브 실행 파일을 생성합니다.",
		"build.title":              "웹 앱 패키징",
		"field.output":             "출력 경로",
		"field.target":             "타겟 플랫폼",
		"field.target.hint":        "Windows는 순수 Go, Linux·macOS는 호스트에 C 툴체인 필요.",
		"target.all":               "전체 플랫폼",
		"target.all.desc":          "Windows·Linux·macOS 한 번에 생성",
		"target.darwin":            "macOS",
		"target.darwin.desc":       "Mach-O 실행 파일(WKWebView)",
		"target.linux":             "Linux",
		"target.linux.desc":        "ELF 실행 파일(WebKit2GTK)",
		"target.windows":           "Windows",
		"target.windows.desc":      "EXE·GUI 서브시스템(WebView2)",
		"status.notReady":          "검증에서 경고 또는 오류 발생 — 아래 참조.",
		"status.ready":             "프로젝트가 빌드 준비되었습니다.",
		"status.building":          "패키징 중…",
		"status.done":              "완료",
		"status.needSource":        "소스 디렉터리를 먼저 입력하세요.",
		"status.needOutput":        "출력 경로를 지정하세요.",
		"status.failed":            "빌드 실패",
		"status.targetPartial":     "일부 타겟 실패 — 아래 참조.",
		"status.targetAllDone":     "요청한 실행 파일을 모두 생성했습니다.",
		"status.buildStarting":     "빌드 시작…",
		"hint.sourceDir":           "index.html 또는 동등한 엔트리 파일 필요.",
		"hint.browse":              "필드에 프로젝트 디렉터리 경로를 직접 입력 또는 붙여넣기.",
		"hint.browseIcon":          ".ico / .png 아이콘 경로를 입력 또는 붙여넣기.",
		"action.browse":            "찾아보기…",
		"action.start":             "開始打包",
		"action.validate":          "검증",
		"footer.note":              "독립 네이티브 앱 생성기 — Windows · Linux · macOS",
	},
}

func main() {
	dir := "internal/i18n/locales"
	for code, adds := range perLocale {
		path := filepath.Join(dir, code+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			os.Exit(1)
		}
		var existing map[string]string
		if err := json.Unmarshal(raw, &existing); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", path, err)
			os.Exit(1)
		}
		// Merge: new keys win, existing others stay.
		for k, v := range adds {
			existing[k] = v
		}
		// Write back sorted by key.
		keys := make([]string, 0, len(existing))
		for k := range existing {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// Build map-of-any with same order won't preserve, so marshal then
		// re-marshal via an ordered approach: build ordered pairs.
		buf, err := marshalOrdered(existing, keys)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal %s: %v\n", path, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, buf, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("updated %s (%d keys)\n", path, len(existing))
	}
}

// marshalOrdered emits JSON with keys in the given order, indented 2 spaces.
func marshalOrdered(m map[string]string, order []string) ([]byte, error) {
	// Use json.Marshal then a manual regex-free approach: marshal each key.
	// Simplest reliable approach: build an ordered slice and marshal an
	// ordered wrapper. encoding/json preserves slice order when marshaling
	// []struct{Key,Value} using json.RawMessage via map[string]json.RawMessage
	// — but simplest is to assemble the string manually.
	out := []byte{'{', '\n'}
	for i, k := range order {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(m[k])
		out = append(out, []byte("  ")...)
		out = append(out, kb...)
		out = append(out, []byte(": ")...)
		out = append(out, vb...)
		if i < len(order)-1 {
			out = append(out, ',')
		}
		out = append(out, '\n')
	}
	out = append(out, '}', '\n')
	return out, nil
}
