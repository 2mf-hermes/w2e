package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	f := filepath.Join("internal", "ui", "assets", "app.js")
	b, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}

	// The file currently has: .replace(/"/g, """)
	// That is bytes: 2F 22 2F 67 2C 20 22 22 22 29
	// The replacement should be: .replace(/"/g, "&quot;")
	// That is bytes: 2F 22 2F 67 2C 20 22 26 71 75 6F 74 3B 22 29

	// needle: the last .replace call including its three quotes + paren
	needle := []byte{0x2F, 0x22, 0x2F, 0x67, 0x2C, 0x20, 0x22, 0x22, 0x22, 0x29}

	// replacement: /" , then "&quot;"
	repl := []byte{0x2F, 0x22, 0x2F, 0x67, 0x2C, 0x20, 0x22, 0x26, 0x71, 0x75, 0x6F, 0x74, 0x3B, 0x22, 0x29}

	n := 0
	for i := 0; i <= len(b)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if b[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			newB := make([]byte, 0, len(b)-len(needle)+len(repl))
			newB = append(newB, b[:i]...)
			newB = append(newB, repl...)
			newB = append(newB, b[i+len(needle):]...)
			b = newB
			n++
			i += len(repl) - 1
		}
	}

	fmt.Printf("Replaced %d occurrences\n", n)
	if err := os.WriteFile(f, b, 0644); err != nil {
		panic(err)
	}

	// Verify
	b2, _ := os.ReadFile(f)
	s2 := string(b2)
	idx := -1
	for i := range s2 {
		if i+11 <= len(s2) && s2[i:i+11] == "escapeHtml" {
			idx = i
			break
		}
	}
	if idx >= 0 {
		end := idx + 200
		if end > len(s2) {
			end = len(s2)
		}
		fmt.Printf("Verify: ...%s\n", s2[idx:end])
	}
}
