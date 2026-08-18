package builder

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/errcode"
	"github.com/minfu/w2e/internal/icon"
)

// hostConfig is the JSON written alongside (or embedded in) the host EXE.
type hostConfig struct {
	AppID     string `json:"app_id"`
	Title     string `json:"title"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Resizable bool   `json:"resizable"`
	EntryFile string `json:"entry_file"`
	Debug     bool   `json:"debug"`
}

// trailerMagic is the sentinel appended after the zip data in the self-extracting EXE.
var trailerMagic = []byte("W2E ZIP1")

// buildFromTemplate builds a Windows EXE using the pre-compiled host template.
// Instead of generating Go source and calling `go build`, this copies the
// embedded host.exe and appends the user's web assets as a zip archive —
// no Go compiler needed at packaging time.
//
// Strategy: write everything to a temp file first, then rename to the final
// output path. This avoids "file in use" errors when overwriting an existing
// running executable.
func (e *Engine) buildFromTemplate(cfg *config.BuildConfig, progress Progress) Result {
	start := time.Now()

	progress("template", "start", "extracting bundled host template", 15)

	// 1. Resolve output path.
	outAbs, err := cfg.ResolveOutputAbs()
	if err != nil {
		return failResult(errcode.OutputNotWritable, "could not resolve output: "+err.Error(), start)
	}
	if err := ensureOutputDir(outAbs); err != nil {
		return failResult(errcode.OutputNotWritable, err.Error(), start)
	}
	// Ensure .exe suffix for Windows.
	if filepath.Ext(outAbs) != ".exe" {
		outAbs += ".exe"
	}

	// Create a temp file in the same directory as the final output
	// (so the rename is atomic on the same volume).
	outDir := filepath.Dir(outAbs)
	tmpFile, err := os.CreateTemp(outDir, "w2e-build-*.exe.tmp")
	if err != nil {
		return failResult(errcode.OutputNotWritable, "could not create temp file: "+err.Error(), start)
	}
	tmpPath := tmpFile.Name()
	// Clean up temp on any error; on success we rename it.
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	// Write the embedded host.exe to the temp file.
	if _, err := tmpFile.Write(hostExeBinary); err != nil {
		tmpFile.Close()
		return failResult(errcode.OutputNotWritable, "could not write host template: "+err.Error(), start)
	}
	progress("template", "ok", "host template extracted", 25)

	// 2. Create a zip containing web assets + config.
	progress("embed", "start", "packaging web assets", 30)

	cfgData := hostConfig{
		AppID:     sanitizeAppID(cfg.AppName),
		Title:     cfg.WindowTitle,
		Width:     cfg.Width,
		Height:    cfg.Height,
		Resizable: cfg.Resizable,
		EntryFile: cfg.EntryFile,
		Debug:     cfg.Debug,
	}
	if cfgData.AppID == "" {
		cfgData.AppID = "w2eapp"
	}
	if cfgData.EntryFile == "" {
		cfgData.EntryFile = "index.html"
	}

	zipBuf := &bytes.Buffer{}
	zw := zip.NewWriter(zipBuf)

	// Write config.
	cfgJSON, _ := json.MarshalIndent(cfgData, "", "  ")
	cfgWriter, err := zw.Create("w2e-config.json")
	if err != nil {
		return failResult(errcode.BuildFailed, "zip create config: "+err.Error(), start)
	}
	if _, err := cfgWriter.Write(cfgJSON); err != nil {
		return failResult(errcode.BuildFailed, "zip write config: "+err.Error(), start)
	}

	// Write web assets.
	if cfg.HasSourceDir() {
		srcAbs, _ := cfg.ResolveSourceAbs()
		if err := addDirToZip(zw, srcAbs, "web"); err != nil {
			return failResult(errcode.BuildFailed, "zip web assets: "+err.Error(), start)
		}
	}
	if err := zw.Close(); err != nil {
		return failResult(errcode.BuildFailed, "zip close: "+err.Error(), start)
	}
	progress("embed", "ok", "web assets packaged", 55)

	// 3. Append zip data + trailer to the temp file.
	progress("patch", "start", "creating self-extracting package", 60)
	zipData := zipBuf.Bytes()
	if _, err := tmpFile.Write(zipData); err != nil {
		tmpFile.Close()
		return failResult(errcode.BuildFailed, "append zip: "+err.Error(), start)
	}
	// Write trailer: [4-byte length LE] [8-byte magic]
	var trailer [12]byte
	binary.LittleEndian.PutUint32(trailer[:4], uint32(len(zipData)))
	copy(trailer[4:], trailerMagic)
	if _, err := tmpFile.Write(trailer[:]); err != nil {
		tmpFile.Close()
		return failResult(errcode.BuildFailed, "write trailer: "+err.Error(), start)
	}
	tmpFile.Close()
	progress("patch", "ok", "self-extracting EXE created", 70)

	// 4. Replace existing file — try several strategies:
	//    a) Remove the old file, then rename temp → final
	//    b) Rename old file to *.old, then rename temp → final
	//    c) As last resort, write temp as filename_N.exe
	removed := false
	if err := os.Remove(outAbs); err == nil {
		removed = true
	}
	if !removed {
		// File may be locked (e.g. running). Try renaming old to .old.
		bakPath := outAbs + ".old"
		_ = os.Remove(bakPath) // remove stale .old if any
		if err := os.Rename(outAbs, bakPath); err == nil {
			removed = true
		}
	}
	if err := os.Rename(tmpPath, outAbs); err == nil {
		progress("patch", "ok", "output file finalized", 73)
	} else if removed {
		// Rename succeeded above but final rename failed — try copy.
		if cpErr := copyFile(tmpPath, outAbs); cpErr != nil {
			// File is locked. Append a number to the filename.
			outAbs = uniqueOutputPath(outAbs)
			_ = os.Rename(tmpPath, outAbs)
			progress("patch", "warn", "output file locked; saved as "+filepath.Base(outAbs), 73)
		} else {
			_ = os.Remove(tmpPath)
			progress("patch", "ok", "output file finalized", 73)
		}
	} else {
		// Could not remove or rename old file. Try unique name.
		outAbs = uniqueOutputPath(outAbs)
		if err := os.Rename(tmpPath, outAbs); err != nil {
			return failResult(errcode.OutputNotWritable, "could not place output: "+err.Error(), start)
		}
		progress("patch", "warn", "output file locked; saved as "+filepath.Base(outAbs), 73)
	}

	// 6. Apply icon if provided.
	if cfg.IconPath != "" {
		progress("icon", "start", "applying icon", 75)
		iconRef, _, perr := icon.Prepare(cfg.IconPath)
		if perr != nil {
			progress("icon", "warn", "icon invalid: "+perr.Error(), 78)
		} else if iconRef.ICOPath != "" {
			iconRef, _ = icon.Apply(outAbs, iconRef)
			if iconRef.Applied {
				progress("icon", "ok", "icon applied", 80)
			} else {
				progress("icon", "warn", "icon could not be applied (rcedit unavailable)", 80)
			}
		}
	}

	// 7. Verify output.
	progress("verify", "start", "verifying output", 85)
	st, err := os.Stat(outAbs)
	if err != nil || st.Size() == 0 {
		return failResult(errcode.VerifyFailed, "output missing or empty", start)
	}
	progress("verify", "ok", "output verified", 95)

	progress("output", "ok", outAbs, 100)

	return Result{
		Success:    true,
		OutputPath: outAbs,
		AppName:    cfg.AppName,
		Target:     "windows",
		Format:     "Windows PE (self-extracting)",
		Size:       st.Size(),
		DurationMs: time.Since(start).Milliseconds(),
	}
}

// addDirToZip recursively adds dir to the zip under prefix.
func addDirToZip(zw *zip.Writer, dir, prefix string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := info.Name()
		if info.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules") {
			return filepath.SkipDir
		}
		zipPath := prefix + "/" + filepath.ToSlash(rel)
		if info.IsDir() {
			zipPath += "/"
			_, err := zw.Create(zipPath)
			return err
		}
		f, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
}

// Ensure we reference the config and errcode packages so they're not flagged unused.
var _ = fmt.Sprintf

// uniqueOutputPath returns a non-existing path by appending _2, _3, etc.
func uniqueOutputPath(p string) string {
	dir := filepath.Dir(p)
	ext := filepath.Ext(p)
	base := strings.TrimSuffix(filepath.Base(p), ext)
	for i := 2; i < 100; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return p // give up after 99 attempts
}
