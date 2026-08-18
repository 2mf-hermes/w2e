package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/minfu/w2e/internal/config"
)

func writeSample(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", full, err)
		}
	}
}

func TestValidateHappyPath(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, map[string]string{
		"index.html": `<!doctype html><html><body><script src="app.js"></script></body></html>`,
		"app.js":     `console.log(1);`,
		"app.css":    `body { color: #000; }`,
	})
	cfg := &config.BuildConfig{SourceDir: dir, EntryFile: "index.html"}
	rep, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !rep.Ready {
		t.Errorf("Ready = false; want true. Errors: %v Warnings: %v", rep.Errors, rep.Warnings)
	}
	if rep.Entry != "index.html" {
		t.Errorf("Entry = %q; want index.html", rep.Entry)
	}
}

func TestValidateRejectsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	// empty
	cfg := &config.BuildConfig{SourceDir: dir}
	_, err := Validate(cfg)
	if err == nil {
		t.Errorf("Validate(empties) err = nil; want error")
	}
}

func TestScanAllSkipNodeModules(t *testing.T) {
	dir := t.TempDir()
	writeSample(t, dir, map[string]string{
		"index.html": `<html></html>`,
		"app.js":     `1;`,
		"node_modules/lib/x.js": `module.exports = 1`,
	})
	files := ScanAll(dir)
	if got, want := len(files.JS), 1; got != want {
		t.Errorf("JS count = %d; want %d (node_modules should be skipped), files=%v", got, want, files.JS)
	}
}
