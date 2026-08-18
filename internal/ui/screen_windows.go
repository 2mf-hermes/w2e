//go:build windows

package ui

import (
	"syscall"
	"unsafe"
)

// screenMetrics returns the primary monitor's width and height in pixels.
func screenMetrics() (w, h int) {
	type point struct{ X, Y int32 }
	type monitorInfo struct {
		CbSize    uint32
		RcMonitor struct {
			Left, Top, Right, Bottom int32
		}
		RcWork struct {
			Left, Top, Right, Bottom int32
		}
		Flags uint32
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	getMonitorW := user32.NewProc("MonitorFromWindow")
	getMonitorInfoW := user32.NewProc("GetMonitorInfoW")

	// Get monitor for the foreground window (0 = primary).
	hWnd, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("GetForegroundWindow").Call()
	mon, _, _ := getMonitorW.Call(hWnd, 2) // MONITOR_DEFAULTTONEAREST

	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	r, _, _ := getMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	if r != 0 {
		w = int(mi.RcMonitor.Right - mi.RcMonitor.Left)
		h = int(mi.RcMonitor.Bottom - mi.RcMonitor.Top)
		return
	}
	// Fallback: SM_CXSCREEN / SM_CYSCREEN
	getSysMetrics := user32.NewProc("GetSystemMetrics")
	cx, _, _ := getSysMetrics.Call(0) // SM_CXSCREEN
	cy, _, _ := getSysMetrics.Call(1) // SM_CYSCREEN
	return int(cx), int(cy)
}
