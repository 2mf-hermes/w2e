package ui

import (
	"github.com/minfu/w2e/internal/browser"
)

// shell opens the given URL in the user's default browser via the
// internal/browser package, which is itself cross-platform.
func shell(url string) error {
	return browser.Open(url)
}
