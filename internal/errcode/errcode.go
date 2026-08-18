// Package errcode defines stable, machine-readable error codes shared between
// the GUI, CLI, and MCP. Codes are part of the public contract and must never
// be reordered or removed; new codes may be appended.
package errcode

// Code is a stable string error code.
type Code string

// Stable error codes. The text-mapping is provided per-message on demand -
// callers receive (Code, human string, machine suggestion string).
const (
	InvalidSource       Code = "INVALID_SOURCE"
	EntryNotFound       Code = "ENTRY_NOT_FOUND"
	InvalidConfig       Code = "INVALID_CONFIG"
	AssetNotFound       Code = "ASSET_NOT_FOUND"
	Webview2Unavailable Code = "WEBVIEW2_NOT_AVAILABLE"
	CompilerNotFound    Code = "COMPILER_NOT_FOUND"
	IconInvalid         Code = "ICON_INVALID"
	OutputNotWritable   Code = "OUTPUT_NOT_WRITABLE"
	BuildFailed         Code = "BUILD_FAILED"
	VerifyFailed        Code = "VERIFY_FAILED"
	UnsupportedOs       Code = "UNSUPPORTED_OS"
	CrossCompileFailed  Code = "CROSS_COMPILE_FAILED"
	NativeToolchain     Code = "NATIVE_TOOLCHAIN_MISSING"
	DebugSymbols        Code = "DEBUG_SYMBOLS_INVALID"
)

// Error carries a stable code alongside its human-readable message. The MCP
// and GUI layers read .Code to emit machine-readable responses, while the
// CLI/UX surfaces .Message and .Suggestion instead.
type Error struct {
	Code       Code
	Message    string
	Suggestion string
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmtWrap(e.Message) + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// New builds an *Error with the supplied fields.
func New(code Code, message string, suggestion string, err error) *Error {
	return &Error{Code: code, Message: message, Suggestion: suggestion, Err: err}
}

func fmtWrap(s string) string { return s }
