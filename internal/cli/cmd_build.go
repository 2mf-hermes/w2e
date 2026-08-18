package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/version"
)

type buildFlags struct {
	entry     string
	name      string
	title     string
	width     int
	height    int
	icon      string
	resizable bool
	output    string
	keepTemp  bool
	url       string
	target    string
}

func cmdVersion(w io.Writer) int {
	info := version.Get()
	fmt.Fprintln(w, "w2e", info.Version, "(build engine "+info.BuildEngine+")")
	fmt.Fprintln(w, "  go:", info.GoVersion)
	fmt.Fprintln(w, "  mcp protocol:", info.MCPProtocol)
	fmt.Fprintln(w, "  webview2 engine:", info.WebView2Engine)
	return 0
}

func cmdBuild(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	// first positional is SOURCE_DIR (unless --url mode); rest are flags
	flags := buildFlags{
		width:     config.DefaultWidth,
		height:    config.DefaultHeight,
		resizable: true,
	}
	parser := argParser{rest: []string{}, out: stderr, head: "build"}
	parser.register(&flagSpec{"--entry", "FILE", &flags.entry})
	parser.register(&flagSpec{"--name", "NAME", &flags.name})
	parser.register(&flagSpec{"--title", "TITLE", &flags.title})
	parser.register(&flagSpec{"--width", "N", &flags.width})
	parser.register(&flagSpec{"--height", "N", &flags.height})
	parser.register(&flagSpec{"--icon", "PATH", &flags.icon})
	parser.register(&flagSpec{"--output", "PATH", &flags.output})
	parser.register(&flagSpec{"--url", "URL", &flags.url})
	parser.register(&flagSpec{"--target", "PLATFORM", &flags.target})
	parser.registerBool("--no-resizable", &flags.resizable, true)
	parser.registerBool("--keep-temp", &flags.keepTemp, false)
	if !parser.parse(args) {
		return 2
	}

	var sourceDir string
	if len(parser.rest) > 0 {
		sourceDir = parser.rest[0]
	}

	cfg := &config.BuildConfig{
		SourceDir:   sourceDir,
		SourceURL:   flags.url,
		EntryFile:   flags.entry,
		OutputFile:  flags.output,
		AppName:     flags.name,
		WindowTitle: flags.title,
		Width:       flags.width,
		Height:      flags.height,
		Resizable:   flags.resizable,
		IconPath:    flags.icon,
		Target:      flags.target,
	}
	eng := NewEngine()
	eng.KeepTemp = flags.keepTemp

	fmt.Fprintf(stdout, "Building..\n")
	progress := func(step, status, msg string, percent int) {
		if status == "ok" || status == "warn" || status == "fail" {
			fmt.Fprintf(stdout, "  [%s] %s\n", status, msg)
		}
	}
	res := eng.Build(cfg, progress)
	if !res.Success {
		// Report every target that failed.
		anyFailed := false
		for _, t := range res.Targets {
			if !t.Success {
				anyFailed = true
				fmt.Fprintf(stderr, "✕ [%s] %s\n", t.Target, t.ErrorMessage)
				if t.ErrorCode != "" {
					fmt.Fprintf(stderr, "  error_code: %s\n", t.ErrorCode)
				}
			}
		}
		if !anyFailed {
			fmt.Fprintf(stderr, "✕ build produced no results\n")
		}
		return 1
	}
	for _, t := range res.Targets {
		if !t.Success {
			fmt.Fprintf(stderr, "✕ [%s] %s\n", t.Target, t.ErrorMessage)
			if t.ErrorCode != "" {
				fmt.Fprintf(stderr, "  error_code: %s\n", t.ErrorCode)
			}
			continue
		}
		fmt.Fprintf(stdout, "✓ %s\n", filepath.Base(t.OutputPath))
		fmt.Fprintf(stdout, "  target: %s\n  format: %s\n  path: %s\n  size: %.2f MB\n  time: %.2fs\n",
			t.Target, t.Format, t.OutputPath,
			float64(t.Size)/1024/1024,
			float64(t.DurationMs)/1000)
	}
	return 0
}

// argParser is a tiny flag parser used internally by CLI commands. It avoids
// pulling in stdlib flag to keep command output stable and predictable.
type argParser struct {
	rest  []string
	out   io.Writer
	head  string
	specs []*flagSpec
	bools  []*boolSpec
}

type flagSpec struct {
	flag   string
	hint   string
	val    interface{} // *string or *int
}

type boolSpec struct {
	flag string
	val  *bool
	cur  bool // current sentinel value (so the parser negates properly)
}

func (a *argParser) register(s *flagSpec) { a.specs = append(a.specs, s) }
func (a *argParser) registerBool(flag string, val *bool, current bool) {
	a.bools = append(a.bools, &boolSpec{flag: flag, val: val, cur: current})
}

func (a *argParser) parse(args []string) bool {
	for i := 0; i < len(args); i++ {
		tok := args[i]
		matched := false
		for _, s := range a.specs {
			if tok == s.flag {
				if i+1 >= len(args) {
					fmt.Fprintf(a.out, "%s: %s requires a %s value\n", a.head, s.flag, s.hint)
					return false
				}
				i++
				val := args[i]
				switch p := s.val.(type) {
				case *string:
					*p = val
				case *int:
					n, err := strconv.Atoi(val)
					if err != nil {
						fmt.Fprintf(a.out, "%s: --width/--height requires an integer, got %q\n", a.head, val)
						return false
					}
					*p = n
				}
				matched = true
				break
			}
		}
		for _, b := range a.bools {
			if tok == b.flag {
				// Toggle relative to default.
				if b.cur {
					*b.val = false
				} else {
					*b.val = true
				}
				matched = true
				break
			}
		}
		if !matched {
			if strings.HasPrefix(tok, "-") {
				fmt.Fprintf(a.out, "%s: unknown flag %s\n", a.head, tok)
				return false
			}
			a.rest = append(a.rest, tok)
		}
	}
	return true
}
