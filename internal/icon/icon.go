// Package icon converts .ico/.png into a Windows icon source embedded into
// the generated host project's go-webview2 require-system.
//
// Background: the Windows .exe icon is set by an *.syso resource file that
// go embeds alongside the *.go sources, OR by go-webview2's WindowOptions
// IconId numeric resource. The simpler, dependency-free approach used here
// is to emit an ico file in the temp project root and write a small *.syso
// generated via a tiny .rc→.syso step. But .rc→.syso needs windres/windres32
// which violates "no CGO/additional toolchain".
//
// Decision: we DON'T bake the icon into the EXE in this build path, because
// both alternatives require either windres (CGO toolchain) or paired .syso
// generation. Instead, the generated EXE ships with the default binary icon,
// and we offer a documented post-build enhancement path using rcedit:
//   rcedit --set-icon app.ico MyApp.exe
// rcedit.exe can be downloaded standalone (MIT) and placed next to w2e.
//
// For today, the builder records the user's icon choice and emits a manifest
// neighbor (assets/.iconref.json) plus a post-build hint; if rcedit is
// reachable on PATH, we apply the icon to the produced EXE non-interactively.
// This keeps the build pure-Go and the EXE production real.
package icon

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Reference is the artifact stored next to the EXE describing the chosen icon.
type Reference struct {
	SourcePath string `json:"source_path"`
	ICOPath    string `json:"ico_path,omitempty"`
	Applied    bool   `json:"applied"`
	AppliedBy  string `json:"applied_by,omitempty"` // "rcedit"
}

// Prepare validates the icon file and (if it's a PNG) writes a sibling ico
// via an embedded conversion routine - but the no-CGO constraint means we
// fall back to a "document and defer to rcedit" approach. Validate first.
//
// Returns the normalized icon reference that the builder stores, plus a
// warning list the GUI/CLI surfaces to the user.
func Prepare(iconPath string) (Reference, []string, error) {
	if iconPath == "" {
		return Reference{}, nil, nil
	}
	ref := Reference{SourcePath: iconPath}
	exists := func(p string) bool {
		st, err := os.Stat(p)
		return err == nil && !st.IsDir()
	}
	if !exists(iconPath) {
		return ref, nil, fmt.Errorf("icon: file not found: %s", iconPath)
	}
	ext := strings.ToLower(filepath.Ext(iconPath))
	var warnings []string
	switch ext {
	case ".ico":
		ref.ICOPath = iconPath
	case ".png":
		// Validate the PNG is actually readable so we can give an immediate error.
		f, err := os.Open(iconPath)
		if err != nil {
			return ref, warnings, fmt.Errorf("icon: cannot open PNG: %w", err)
		}
		defer f.Close()
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return ref, warnings, fmt.Errorf("icon: invalid PNG: %w", err)
		}
		if cfg.Width > 256 || cfg.Height > 256 {
			warnings = append(warnings, fmt.Sprintf(
				"icon PNG is %dx%d; Windows sizes icons to 256x256 max, consider downsizing",
				cfg.Width, cfg.Height))
		}
		// We can't convert PNG→ICO without extra libs in pure-Go easily; we mark
		// the user's PNG as-is and let rcedit (which accepts PNG directly via
		// --set-icon) handle it.
		ref.ICOPath = iconPath
		warnings = append(warnings,
			"PNG icon will be applied by rcedit if available; otherwise the default EXE icon is used")
	default:
		return ref, warnings, fmt.Errorf("icon: unsupported extension %q (use .ico or .png)", ext)
	}
	return ref, warnings, nil
}

// Apply attempts to set the icon on the EXE using rcedit if reachable.
// Returns the (possibly updated) Reference and any error. If rcedit is
// absent, Apply is a no-op (ref.Applied stays false) and returns nil error
// to keep the build successful.
func Apply(exePath string, ref Reference) (Reference, error) {
	if ref.ICOPath == "" {
		return ref, nil
	}
	rc, err := exec.LookPath("rcedit.exe")
	if err != nil {
		return ref, nil // not installed; non-fatal
	}
	cmd := exec.Command(rc, exePath, "--set-icon", ref.ICOPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return ref, fmt.Errorf("icon apply failed: %w: %s", err, string(out))
	}
	ref.Applied = true
	ref.AppliedBy = "rcedit"
	return ref, nil
}

// Persist writes the Reference next to the EXE so users can re-apply later.
func Persist(exeDir string, ref Reference) error {
	if ref.SourcePath == "" {
		return nil
	}
	data, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		return errors.New("icon: marshal failed")
	}
	return os.WriteFile(filepath.Join(exeDir, "w2e-icon.json"), data, 0o644)
}
