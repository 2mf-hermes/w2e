// Package i18n provides translation lookup for w2e UI and CLI messages.
//
// Locales are embedded via //go:embed and selected at runtime from the
// system language (or a user override). The lookup is a flat key->string
// map with a stable fallback chain (current → en). The empty-string key
// returns "" safely so partially-translated UIs don't crash.
//
// Supported locales (spec §27): zh-TW, zh-CN, en, ja, ko.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

// Locale holds a single language's string map.
type Locale struct {
	Code string
	data map[string]string
}

// Bundle is the runtime translator.
type Bundle struct {
	mu      sync.RWMutex
	cur     *Locale
	fallback *Locale
	loaded  map[string]*Locale
}

var defaultBundle *Bundle
var defaultOnce sync.Once

// Default returns a process-wide Bundle initialized with the detected
// system language (or English fallback).
func Default() *Bundle {
	defaultOnce.Do(func() {
		b, err := NewBundle(Detect())
		if err != nil {
			b, _ = NewBundle("en")
		}
		defaultBundle = b
	})
	return defaultBundle
}

// NewBundle builds a Bundle seeded from the given locale code (e.g. "zh-TW").
// Missing locales fall back to English. All bundled locales are loaded
// eagerly so Available() reflects the full set shipped with w2e.
func NewBundle(loc string) (*Bundle, error) {
	b := &Bundle{loaded: map[string]*Locale{}}
	if err := b.load("en"); err != nil {
		return nil, err
	}
	b.fallback = b.loaded["en"]
	// eagerly load the rest so the language switcher can show them.
	for _, code := range []string{"zh-TW", "zh-CN", "ja", "ko"} {
		_ = b.load(code)
	}
	if err := b.Set(loc); err != nil {
		// best-effort: ignore the failure and fall back to English silently.
		b.cur = b.fallback
	}
	return b, nil
}

// Set changes the active locale. Unknown codes -> English.
func (b *Bundle) Set(code string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if code == "" {
		b.cur = b.fallback
		return nil
	}
	if existing, ok := b.loaded[code]; ok {
		b.cur = existing
		return nil
	}
	if err := b.load(code); err != nil {
		// English fallback.
		return err
	}
	b.cur = b.loaded[code]
	return nil
}

// Active returns the current locale code.
func (b *Bundle) Active() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.cur == nil {
		return "en"
	}
	return b.cur.Code
}

// Locale is an alias for Active() used by the UI package.
func (b *Bundle) Locale() string { return b.Active() }

// SetLocale is an alias for Set used by the UI package.
func (b *Bundle) SetLocale(code string) error { return b.Set(code) }

// Available lists the locales that are currently loaded (the UI shows them
// in its language switcher).
func (b *Bundle) Available() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	codes := make([]string, 0, len(b.loaded))
	for code := range b.loaded {
		codes = append(codes, code)
	}
	return codes
}

// All returns the merged translation map of the active locale over the
// English fallback (so partially-translated UIs still get every key).
func (b *Bundle) All() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := map[string]string{}
	if b.fallback != nil {
		for k, v := range b.fallback.data {
			out[k] = v
		}
	}
	if b.cur != nil {
		for k, v := range b.cur.data {
			if v != "" {
				out[k] = v
			}
		}
	}
	return out
}

// Keys returns the merged translation keys (caller-visible subset).
func (b *Bundle) Keys() []string {
	m := b.All()
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// load reads a locale file by code.
func (b *Bundle) load(code string) error {
	data, err := localesFS.ReadFile("locales/" + code + ".json")
	if err != nil {
		// some locales are multi-segment (zh-TW) — but embedded name is exactly code.json
		return fmt.Errorf("i18n: locale %q not found: %w", code, err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("i18n: locale %q invalid JSON: %w", code, err)
	}
	b.loaded[code] = &Locale{Code: code, data: m}
	return nil
}

// T translates a key. Optional args fill the {0},{1}... placeholders via fmt.
func (b *Bundle) T(key string, args ...any) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var str string
	if b.cur != nil {
		if s, ok := b.cur.data[key]; ok && s != "" {
			str = s
		}
	}
	if str == "" && b.fallback != nil {
		if s, ok := b.fallback.data[key]; ok && s != "" {
			str = s
		}
	}
	if str == "" {
		// stable dev-time marker so untranslated keys are visible but
		// recognizable instead of quietly dropping to "".
		return "[" + key + "]"
	}
	if len(args) > 0 {
		return fmt.Sprintf(str, args...)
	}
	return str
}

// Detect probes the OS environment for a supported language code.
// Order: LANGUAGE > LC_ALL > LC_MESSAGES > LANG then Windows language UI
// (best-effort, no Go stdlib helper for GetUserDefaultLocaleName).
// Unsupported -> "en".
func Detect() string {
	for _, env := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES"} {
		if v := os.Getenv(env); v != "" {
			if code := normalizeLocale(v); code != "" {
				return code
			}
		}
	}
	if lang := os.Getenv("LANG"); lang != "" {
		if code := normalizeLocale(lang); code != "" {
			return code
		}
	}
	return defaultForWindows()
}

// normalizeLocale turns an OS locale name into one of our supported codes:
// "zh-TW", "zh-CN", "en", "ja", "ko". It accepts POSIX ("zh_TW.UTF-8"),
// Unix env ("zh_TW"), and Windows BCP-47 ("zh-TW", "en-US", "ja", "ko").
// Returns "" when the language isn't one we support.
func normalizeLocale(lang string) string {
	lang = strings.TrimSuffix(strings.TrimSpace(lang), "\"")
	if lang == "" || lang == "C" || lang == "POSIX" {
		return ""
	}
	// Strip .UTF-8/@modifier; then split region separator (- or _).
	core := strings.SplitN(strings.SplitN(lang, ".", 2)[0], "@", 2)[0]
	core = strings.ToLower(strings.TrimSpace(core))
	if core == "" {
		return ""
	}
	parts := strings.SplitN(strings.ReplaceAll(core, "_", "-"), "-", 3)
	primary := parts[0]
	region := ""
	if len(parts) >= 2 {
		region = parts[1]
	}
	switch primary {
	case "zh":
		switch region {
		case "cn", "sg":
			return "zh-CN"
		case "tw", "hk", "mo":
			return "zh-TW"
		default:
			return "zh-TW" // bare zh, "zh-hans", "zh-hant" -> Traditional default
		}
	case "en":
		return "en"
	case "ja":
		return "ja"
	case "ko":
		return "ko"
	}
	return ""
}

// defaultForWindows reads the Windows user UI language via the
// platform-specific windows_lang.go (GetUserDefaultLocaleName) and maps it
// to one of our supported i18n codes. On non-Windows, or when detection
// fails, it falls back to "en". See winDetectWindowsLang (build-tagged).
func defaultForWindows() string {
	if code := winDetectWindowsLang(); code != "" {
		return code
	}
	return "en"
}
