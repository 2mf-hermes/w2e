// Quick patch: add two missing locale keys across all 5 locales.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var adds = map[string]map[string]string{
	"en":    {"build.title.window": "Window", "status.validating": "Validating…"},
	"zh-TW": {"build.title.window": "視窗", "status.validating": "驗證中…"},
	"zh-CN": {"build.title.window": "窗口", "status.validating": "验证中…"},
	"ja":    {"build.title.window": "ウィンドウ", "status.validating": "検証中…"},
	"ko":    {"build.title.window": "윈도우", "status.validating": "검증 중…"},
}

func main() {
	dir := "internal/i18n/locales"
	for code, kv := range adds {
		path := filepath.Join(dir, code+".json")
		raw, err := os.ReadFile(path)
		if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
		for k, v := range kv { m[k] = v }
		keys := make([]string, 0, len(m)); for k := range m { keys = append(keys, k) }
		sort.Strings(keys)
		out := []byte{'{', '\n'}
		for i, k := range keys {
			kb, _ := json.Marshal(k); vb, _ := json.Marshal(m[k])
			out = append(out, []byte("  ")...); out = append(out, kb...); out = append(out, []byte(": ")...); out = append(out, vb...)
			if i < len(keys)-1 { out = append(out, ',') }
			out = append(out, '\n')
		}
		out = append(out, '}', '\n')
		if err := os.WriteFile(path, out, 0o644); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	}
}
