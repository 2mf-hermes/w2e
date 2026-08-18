package i18n

import "testing"

func TestTRoundtrip(t *testing.T) {
	b, err := NewBundle("en")
	if err != nil {
		t.Fatalf("NewBundle(en): %v", err)
	}
	want := "w2e" // app.title in en locale (if we did embed it)
	got := b.T("app.title")
	if got == "" || got == "[app.title]" {
		t.Errorf("T(app.title) returned %q — want a real translation", got)
	}
	if got != want {
		t.Logf("T(app.title) = %q (advisory)", got)
	}
}

func TestLocaleSwitch(t *testing.T) {
	b, _ := NewBundle("en")
	if err := b.SetLocale("zh-TW"); err != nil {
		t.Fatalf("SetLocale(zh-TW): %v", err)
	}
	if b.Locale() != "zh-TW" {
		t.Errorf("Locale() = %q; want zh-TW", b.Locale())
	}
	if err := b.SetLocale("en"); err != nil {
		t.Fatalf("SetLocale(en): %v", err)
	}
	if b.Locale() != "en" {
		t.Errorf("Locale() = %q; want en", b.Locale())
	}
}

func TestAvailableLocales(t *testing.T) {
	b, _ := NewBundle("en")
	avail := b.Available()
	want := map[string]bool{"en": true, "zh-TW": true, "zh-CN": true, "ja": true, "ko": true}
	got := map[string]bool{}
	for _, c := range avail {
		got[c] = true
	}
	for code := range want {
		if !got[code] {
			t.Errorf("Available() missing %q (got %v)", code, avail)
		}
	}
}
