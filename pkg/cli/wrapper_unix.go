//go:build !windows
// +build !windows

package cli

import (
	"syscall"
	"unsafe"
)

// getTerminalSize returns the terminal dimensions on Unix systems
func (w *Wrapper) getTerminalSize() (width, height int) {
	// Use syscall to get terminal size
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 || errno != 0 {
		// Default size if we can't detect
		return 80, 24
	}

	return int(ws.Col), int(ws.Row)
}
