// verify-ui-keys reads the w2e UI assets and locale JSON, then asserts:
//  1. all [data-i18n="..."] keys appearing in index.html also appear in every
//     locale (en, zh-TW, zh-CN, ja, ko); and
//  2. all t("key", "..") lookups in app.js appear in every locale.
// Exits non-zero on any missing key, so the build can gate on this check.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	uiDir := "internal/ui/assets"
	locDir := "internal/i18n/locales"
	codes := []string{"en", "zh-TW", "zh-CN", "ja", "ko"}

	// 1. Collect referenced keys.
	htmlBytes, err := os.ReadFile(filepath.Join(uiDir, "index.html"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	jsBytes, err := os.ReadFile(filepath.Join(uiDir, "app.js"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	html := string(htmlBytes)
	js := string(jsBytes)

	keys := map[string]bool{}
	htmlRe := regexp.MustCompile(`data-i18n="([^"]+)"`)
	for _, m := range htmlRe.FindAllStringSubmatch(html, -1) {
		keys[m[1]] = true
	}
	jsT := regexp.MustCompile(`t\(\s*"([^"]+)"\s*,`)
	for _, m := range jsT.FindAllStringSubmatch(js, -1) {
		keys[m[1]] = true
	}

	// 2. Load all locales and check.
	missingByLocale := map[string][]string{}
	for _, code := range codes {
		path := filepath.Join(locDir, code+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
		var loc map[string]string
		if err := json.Unmarshal(raw, &loc); err != nil {
			fmt.Fprintf(os.Stderr, "%s parse: %v\n", path, err)
			os.Exit(1)
		}
		for k := range keys {
			if v := loc[k]; strings.TrimSpace(v) == "" {
				missingByLocale[code] = append(missingByLocale[code], k)
			}
		}
	}

	// 3. Report.
	totalMissing := 0
	codes2 := make([]string, 0, len(missingByLocale))
	for c := range missingByLocale { codes2 = append(codes2, c) }
	sort.Strings(codes2)
	for _, c := range codes2 {
		ks := missingByLocale[c]
		sort.Strings(ks)
		fmt.Fprintf(os.Stderr, "[%s] missing %d keys: %v\n", c, len(ks), ks)
		totalMissing += len(ks)
	}
	fmt.Printf("referenced keys=%d, locales=%d, total missing=%d\n", len(keys), len(codes), totalMissing)
	if totalMissing > 0 {
		os.Exit(1)
	}
}
