// fix-escape: writes the proper HTML-entity bytes into app.js's escapeHtml
// using explicit hex rune concatenation so no layer of the toolchain can
// collapse the entities back to bare characters. Idempotent.
package main

import (
	"os"
	"strings"
)

func main() {
	path := "internal/ui/assets/app.js"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	s := string(b)

	wrongAmp := ".replace(/&/g, \"&\")"
	wrongLt := ".replace(/</g, \"<\")"
	wrongGt := ".replace(/>/g, \">\")"

	rightAmp := ".replace(/&/g, \"" + ampEntity() + "\")"
	rightLt := ".replace(/</g, \"" + ltEntity() + "\")"
	rightGt := ".replace(/>/g, \"" + gtEntity() + "\")"

	if !strings.Contains(s, wrongAmp) {
		println("amp pattern not found - maybe already patched")
		return
	}
	s = strings.ReplaceAll(s, wrongAmp, rightAmp)
	s = strings.ReplaceAll(s, wrongLt, rightLt)
	s = strings.ReplaceAll(s, wrongGt, rightGt)

	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		panic(err)
	}
	println("fixed: escapeHtml now uses proper HTML entities")
}

// Entities as explicit byte sequences so that this source file itself cannot
// accidentally carry the collapsed forms.
func ampEntity() string { return string([]byte{0x26, 0x61, 0x6d, 0x70, 0x3b}) } // &
func ltEntity() string  { return string([]byte{0x26, 0x6c, 0x74, 0x3b}) }        // <
func gtEntity() string  { return string([]byte{0x26, 0x67, 0x74, 0x3b}) }        // >
