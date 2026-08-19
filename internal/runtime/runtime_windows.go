//go:build windows

package runtime

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// resident-client GUID constant per Microsoft Edge WebView2 SDK.
//
// The Microsoft-documented Evergreen WebView2 Runtime EdgeUpdate client GUID is
//
//	{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}
//
// (note the suffix A7E4C5, NOT A5E36C — the latter is a stale/typo GUID that
// does not match any real EdgeUpdate client and caused false "not installed"
// reports even on machines with the runtime present).
const residentClient = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

// Detect probes the host for the WebView2 Runtime. Returns not-found if
// unavailable; it never panics and never elevates.
func Detect() Detection {
	// 1) System Evergreen runtime - check both CURRENT_USER and LOCAL_MACHINE
	//    so that we honor per-user and per-machine installs.
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, base := range []string{
			`SOFTWARE\Microsoft\EdgeUpdate\Clients\` + residentClient,
			`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + residentClient,
		} {
			k, err := registry.OpenKey(root, base, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			v, _, _ := k.GetStringValue("pv")
			_ = k.Close()
			if v != "" && v != "0.0.0.0" {
				return Detection{
					Kind:    KindSystem,
					Version: v,
					Source:  "Microsoft WebView2 Runtime (Evergreen; system)",
				}
			}
		}
	}

	// 2) App-local fixed-version bundle directly next to w2e host or produced EXE.
	if d := detectFixedBundle(); d.Kind == KindFixedBundle {
		return d
	}

	return Detection{Kind: KindNotFound, Source: "not installed"}
}

// detectFixedBundle searches for an embedded fixed-version runtime beside the
// running EXE.
func detectFixedBundle() Detection {
	exe, err := os.Executable()
	if err != nil {
		return Detection{Kind: KindNotFound}
	}
	dir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(dir, "runtime"),
		filepath.Join(dir, "Microsoft.WebView2.FixedVersionRuntime.64-bit"),
	}
	for _, c := range candidates {
		// A fixed-version bundle contains msedge.exe; presence is the marker.
		if st, err := os.Stat(filepath.Join(c, "msedge.exe")); err == nil && !st.IsDir() {
			subdirs := listVersionSubdirs(c)
			ver := ""
			if len(subdirs) > 0 {
				ver = subdirs[0]
			}
			return Detection{
				Kind:    KindFixedBundle,
				Version: ver,
				Path:    c,
				Source:  "App-local fixed-version WebView2 runtime",
			}
		}
	}
	return Detection{Kind: KindNotFound}
}

// listVersionSubdirs returns directory names resembling "1.2.3.4" forms.
func listVersionSubdirs(parent string) []string {
	var out []string
	entries, err := os.ReadDir(parent)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.Contains(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
}
