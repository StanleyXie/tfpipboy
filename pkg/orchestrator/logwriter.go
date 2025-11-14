package orchestrator

import (
	"io"
)

// CleanLogWriter wraps an io.Writer and strips ANSI escape codes
// This ensures log files are readable without color codes
type CleanLogWriter struct {
	writer io.Writer
}

// NewCleanLogWriter creates a new clean log writer
func NewCleanLogWriter(writer io.Writer) *CleanLogWriter {
	return &CleanLogWriter{
		writer: writer,
	}
}

// Write implements io.Writer, stripping ANSI codes before writing
func (w *CleanLogWriter) Write(p []byte) (n int, err error) {
	// Strip ANSI codes from the input
	cleaned := StripANSI(string(p))

	// Write the cleaned content
	_, err = w.writer.Write([]byte(cleaned))
	if err != nil {
		return 0, err
	}

	// Return the original length (what was passed in)
	return len(p), nil
}

// MultiWriter creates a writer that duplicates writes to multiple writers
// Similar to io.MultiWriter but allows different processing per writer
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a new multi-writer
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write implements io.Writer
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

// TeeWriter creates a writer that writes to both stdout (with colors) and log file (without colors)
func TeeWriter(stdout io.Writer, logFile io.Writer) io.Writer {
	return NewMultiWriter(
		stdout,                     // Terminal output (keeps ANSI colors)
		NewCleanLogWriter(logFile), // Log file (strips ANSI colors)
	)
}
