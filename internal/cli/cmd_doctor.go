package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	stdruntime "runtime"

	"github.com/minfu/w2e/internal/builder"
	w2eruntime "github.com/minfu/w2e/internal/runtime"
)

func cmdDoctor(stdout io.Writer) int {
	fmt.Fprintln(stdout, "w2e doctor — environment diagnostics")
	fmt.Fprintln(stdout, "")

	dr := builder.Doctor()

	// OS / arch
	emit(stdout, fmt.Sprintf("%s %s", osName(stdruntime.GOOS), archName(stdruntime.GOARCH)),
		okOrWarn(stdruntime.GOOS == "windows" || stdruntime.GOOS == "linux" || stdruntime.GOOS == "darwin"))

	// Go toolchain
	emit(stdout, "Go toolchain", okOrWarn(dr.Go))

	// WebView2 Runtime detection
	det := w2eruntime.Detect()
	if stdruntime.GOOS == "windows" {
		emit(stdout, "WebView2 Runtime", okOrWarn(det.Kind != w2eruntime.KindNotFound)+" ("+det.Source+")")
	}

	// Temp and output writable
	if t, err := os.MkdirTemp(os.TempDir(), "w2e-doctor-"); err == nil {
		_ = os.RemoveAll(t)
		emit(stdout, "Temporary Directory", "✓ "+os.TempDir())
	} else {
		emit(stdout, "Temporary Directory", "✕ not writable")
	}

	// Data folder
	if dr.Permissions {
		emit(stdout, "User data folder", "✓")
	} else {
		emit(stdout, "User data folder", "✕ not writable")
	}

	// Cross-platform build targets
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Build targets:")
	for _, ct := range dr.CrossTargets {
		marker := okOrWarn(ct.Go && ct.CC)
		if ct.Cross {
			fmt.Fprintf(stdout, "  %-22s %s cross-compile (Go=%v / C=%v)\n",
				string(ct.Target)+":", marker, ct.Go, ct.CC)
		} else {
			fmt.Fprintf(stdout, "  %-22s %s native (Go=%v / C=%v)\n",
				string(ct.Target)+":", marker, ct.Go, ct.CC)
		}
		if ct.Notes != "" {
			fmt.Fprintf(stdout, "    %s\n", ct.Notes)
		}
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintf(stdout, "build_available: %v\n", dr.BuildAvailable)
	// overall cross-target summary
	anyCross := false
	for _, ct := range dr.CrossTargets {
		if ct.Cross && ct.Go && ct.CC {
			anyCross = true
			break
		}
	}
	fmt.Fprintf(stdout, "cross_compile_available: %v\n", anyCross)
	return 0
}

func emit(w io.Writer, name, val string) {
	fmt.Fprintf(w, "  %-20s %s\n", name+":", val)
}

func okOrWarn(ok bool) string {
	if ok {
		return "✓"
	}
	return "✕"
}

func goReachesOnPath() bool {
	binName := "go"
	if stdruntime.GOOS == "windows" {
		binName = "go.exe"
	}
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == "" {
			continue
		}
		cand := filepath.Join(p, binName)
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func osName(os string) string {
	switch os {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	}
	return os
}

func archName(a string) string {
	switch a {
	case "amd64":
		return "x64"
	case "arm64":
		return "ARM64"
	}
	return a
}
