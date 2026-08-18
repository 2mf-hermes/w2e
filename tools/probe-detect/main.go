// standalone probe of i18n.Detect on this Windows machine.
package main

import (
	"fmt"

	"github.com/minfu/w2e/internal/i18n"
)

func main() {
	code := i18n.Detect()
	fmt.Printf("Detected locale code: %q\n", code)
	b := i18n.Default()
	fmt.Printf("Bundle active: %q\n", b.Active())
	fmt.Printf("app.title = %q\n", b.T("app.title"))
}
