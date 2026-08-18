// Package webserver / mime.go — MIME-type table used by the embedded server.
// Spec §11 mandates correct MIME handling; bad MIME = broken modules.
package webserver

import (
	"strings"
)

// mimeTypeFor returns the canonical Content-Type for an extension.
// Falls back to octet-stream so mispackaged files still download.
func mimeTypeFor(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	// HTML / text
	case "", "html", "htm":
		return "text/html; charset=utf-8"
	case "css":
		return "text/css; charset=utf-8"
	// scripts: module & non-module variants
	case "js", "mjs":
		return "application/javascript; charset=utf-8"
	case "cjs":
		return "text/javascript; charset=utf-8"
	case "json":
		return "application/json; charset=utf-8"
	case "map":
		return "application/json; charset=utf-8"
	case "txt":
		return "text/plain; charset=utf-8"
	case "xml":
		return "application/xml; charset=utf-8"
	case "svg":
		return "image/svg+xml"
	// images
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	case "avif":
		return "image/avif"
	case "bmp":
		return "image/bmp"
	// fonts
	case "woff":
		return "font/woff"
	case "woff2":
		return "font/woff2"
	case "ttf":
		return "font/ttf"
	case "otf":
		return "font/otf"
	case "eot":
		return "application/vnd.ms-fontobject"
	// media
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mp3":
		return "audio/mpeg"
	case "ogg":
		return "audio/ogg"
	case "wav":
		return "audio/wav"
	case "wasm":
		return "application/wasm"
	// misc
	case "pdf":
		return "application/pdf"
	}
	return "application/octet-stream"
}
