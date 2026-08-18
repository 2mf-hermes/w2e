//go:build windows

// ui_dialog_windows.go: native Windows file / folder pickers for the w2e GUI
// via syscall (comdlg32 GetOpenFileNameW for files, shell32 SHBrowseForFolderW
// for folders). No external dependency, no CGO.
//
// The hwndOwner parameter is the WebView2 window handle, used to parent the
// native dialog so it stays on top and returns focus correctly.  We deliberately
// do NOT call CoInitializeEx / CoUninitialize — WebView2 already owns the COM
// apartment on this thread; doing it ourselves would tear it down.
package ui

import (
	"syscall"
	"unsafe"
)

var (
	comdlg32        = syscall.NewLazyDLL("comdlg32.dll")
	shell32         = syscall.NewLazyDLL("shell32.dll")
	ole32           = syscall.NewLazyDLL("ole32.dll")
	procGetOpenFile = comdlg32.NewProc("GetOpenFileNameW")
	procSHBrowse    = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPath   = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskFree  = ole32.NewProc("CoTaskMemFree")
)

// listString builds a Win32 open-files filter list. Each (desc, spec) becomes
// a NUL-terminated UTF-16 segment; the list ends with an extra NUL.
func listString(pairs ...[2]string) []uint16 {
	var out []uint16
	for _, p := range pairs {
		for _, s := range p {
			out = append(out, utf16FromASCII(s)...)
		}
	}
	out = append(out, 0) // final list terminator
	return out
}

// utf16FromASCII encodes an ASCII Go string to UTF-16 terminated by 0.
func utf16FromASCII(s string) []uint16 {
	out := make([]uint16, 0, len(s)+1)
	for i := 0; i < len(s); i++ {
		out = append(out, uint16(s[i]))
	}
	return append(out, 0)
}

// pickFile shows a Windows "Open" dialog restricted to common icon formats
// the builder supports (.ico / .png).  hwndOwner is the WebView2 window.
func pickFile(hwndOwner uintptr) string {
	// OPENFILENAMEW — all pointer-sized fields must be pointers, NOT inline
	// buffers.  Field order and alignment must match the Win32 struct exactly.
	// See https://learn.microsoft.com/en-us/windows/win32/api/commdlg/ns-commdlg-openfilenamew
	type openFileNameW struct {
		StructSize      uint32
		_pad0           [4]byte // align hwndOwner to 8
		HwndOwner       uintptr
		Instance        uintptr
		Filter          *uint16
		CustomFilter    *uint16
		MaxCustFilter   uint32
		FilterIndex     uint32
		File            *uint16   // LPWSTR — pointer to buffer
		MaxFile         uint32
		_pad1           [4]byte // align lpstrFileTitle to 8
		FileTitle       *uint16   // LPWSTR — pointer to buffer
		MaxFileTitle    uint32
		_pad2           [4]byte // align lpstrInitialDir to 8
		InitialDir      *uint16
		Title           *uint16
		Flags           uint32
		nFileOffset     uint16
		nFileExtension  uint16
		DefExt          *uint16
		CustData        uintptr
		HookProc        uintptr
		TemplateName    *uint16
		pvReserved      uintptr // Vista+: reserved, must be NULL
		dwReserved      uint32  // Vista+: reserved, must be 0
		FlagsEx         uint32  // Vista+: extended flags
	}

	// Allocate buffers that outlive the dialog call.
	var fileBuf [1024]uint16
	var titleBuf [512]uint16
	for i := range fileBuf {
		fileBuf[i] = 0
	}
	for i := range titleBuf {
		titleBuf[i] = 0
	}

	filter := listString([2]string{"Icon", "*.ico;*.png"}, [2]string{"All files", "*.*"})
	dlgTitle := utf16FromASCII("Choose an icon")

	var ofn openFileNameW
	ofn.StructSize = uint32(unsafe.Sizeof(ofn))
	ofn.HwndOwner = hwndOwner
	ofn.Filter = &filter[0]
	ofn.File = &fileBuf[0]
	ofn.MaxFile = uint32(len(fileBuf))
	ofn.FileTitle = &titleBuf[0]
	ofn.MaxFileTitle = uint32(len(titleBuf))
	ofn.Title = &dlgTitle[0]
	// OFN_PATHMUSTEXIST | OFN_FILEMUSTEXIST | OFN_HIDEREADONLY | OFN_NOCHANGEDIR
	ofn.Flags = 0x00000800 | 0x00001000 | 0x00000004 | 0x00000008

	r, _, _ := procGetOpenFile.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(fileBuf[:])
}

// pickDirectory shows a Windows folder-browse dialog and returns the chosen
// directory.  hwndOwner is the WebView2 window.  On cancel or failure returns "".
func pickDirectory(hwndOwner uintptr) string {
	// BROWSEINFOW — pszDisplayName is a POINTER (LPWSTR), not an inline buffer.
	// See https://learn.microsoft.com/en-us/windows/win32/api/shlobj_core/ns-shlobj_core-browseinfow
	type browseInfoW struct {
		Hwnd     uintptr
		Root     *uint16
		DispName *uint16  // LPWSTR — pointer to [MAX_PATH]uint16 buffer
		Title    *uint16
		Flags    uint32
		_pad     [4]byte // align Callback to 8
		Callback uintptr
		Param    uintptr
		Image    int32
	}

	var dispNameBuf [260]uint16
	title := utf16FromASCII("Pick a web project directory")

	var bi browseInfoW
	bi.Hwnd = hwndOwner
	bi.Title = &title[0]
	bi.DispName = &dispNameBuf[0]
	// BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE | BIF_EDITBOX
	bi.Flags = 0x00000001 | 0x00000100 | 0x00000010

	pidl, _, _ := procSHBrowse.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskFree.Call(pidl)

	var path [1024]uint16
	r, _, _ := procSHGetPath.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(path[:])
}

// utf16FromString encodes an ASCII Go string to UTF-16 terminated by 0,
// used by callers that build single-segment buffers (titles etc).
func utf16FromString(s string) []uint16 {
	return utf16FromASCII(s)
}
