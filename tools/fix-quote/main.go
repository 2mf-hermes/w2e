package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	f := filepath.Join("internal", "ui", "assets", "app.js")
	b, err := os.ReadFile(f)
	if err != nil { panic(err) }
	s := string(b)

	// Build "&quot;" from hex bytes to prevent toolchain entity collapsing
	qt := string([]byte{0x26, 0x71, 0x75, 0x6F, 0x74, 0x3B}) // &quot;

	old := `.replace(/"/g, "")`
	new := `.replace(/"/g, "` + qt + `")`

	if s == "" { panic("empty file") }
	n := 0
	for i := 0; i < len(s); i++ {
		// find old pattern
		idx := indexOf(s[i:], old)
		if idx < 0 { break }
		s = s[:i+idx] + new + s[i+idx+len(old):]
		n++
		i += idx + len(new)
	}
	if n == 0 {
		fmt.Println("WARN: old pattern not found")
	} else {
		fmt.Printf("Fixed %d occurrences\n", n)
	}
	if err := os.WriteFile(f, []byte(s), 0644); err != nil { panic(err) }
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub { return i }
	}
	return -1
}
