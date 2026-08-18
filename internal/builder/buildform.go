package builder

import (
	"github.com/minfu/w2e/internal/config"
)

// BuildForm is the JSON struct posted by the GUI's "開始打包" form. It
// mirrors config.BuildConfig but keeps booleans explicit (not omitempty)
// so the form roundtrips without surprising "true even when omitted" bugs.
type BuildForm struct {
	SourceDir   string `json:"source_dir"`
	Entry       string `json:"entry,omitempty"`
	Output      string `json:"output"`
	AppName     string `json:"app_name,omitempty"`
	WindowTitle string `json:"window_title,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Resizable   bool   `json:"resizable"`
	Icon        string `json:"icon,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	Target      string `json:"target,omitempty"`
	KeepTemp    bool   `json:"keep_temp,omitempty"`
}

// ToBuildConfig converts the form into a normalized BuildConfig.
func (f *BuildForm) ToBuildConfig() *config.BuildConfig {
	cfg := &config.BuildConfig{
		SourceDir:   f.SourceDir,
		EntryFile:   f.Entry,
		OutputFile:  f.Output,
		AppName:     f.AppName,
		WindowTitle: f.WindowTitle,
		Width:       f.Width,
		Height:      f.Height,
		Resizable:   f.Resizable,
		IconPath:    f.Icon,
		SourceURL:   f.SourceURL,
		Target:      f.Target,
	}
	if cfg.Width == 0 {
		cfg.Width = config.DefaultWidth
	}
	if cfg.Height == 0 {
		cfg.Height = config.DefaultHeight
	}
	if cfg.WindowTitle == "" {
		cfg.WindowTitle = config.DefaultWindowTitle
	}
	return cfg
}
