//go:build windows

package cli

// getTerminalSize returns the terminal dimensions on Windows
func (w *Wrapper) getTerminalSize() (width, height int) {
	// On Windows, return default terminal size
	// Could use Windows API (kernel32.dll GetConsoleScreenBufferInfo) but
	// for now we'll use sensible defaults
	return 80, 24
}
