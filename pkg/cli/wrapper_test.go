package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWrapper(t *testing.T) {
	w := NewWrapper()
	if w == nil {
		t.Fatal("NewWrapper() returned nil")
	}

	if w.authManager == nil {
		t.Error("authManager not initialized")
	}

	if w.tfManager == nil {
		t.Error("tfManager not initialized")
	}

	if w.commandHistory == nil {
		t.Error("commandHistory not initialized")
	}

	if w.commandTimeout != defaultCommandTimeout {
		t.Errorf("commandTimeout = %v, want %v", w.commandTimeout, defaultCommandTimeout)
	}

	if w.historyFile == "" {
		t.Error("historyFile not set")
	}
}

func TestSetupHistoryFile(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "tfpipboy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w := NewWrapper()
	w.historyFile = filepath.Join(tmpDir, ".tfpipboy_history")

	// Test history file creation
	if err := w.setupHistoryFile(); err != nil {
		t.Fatalf("setupHistoryFile() error = %v", err)
	}

	// Verify file exists
	info, err := os.Stat(w.historyFile)
	if err != nil {
		t.Fatalf("History file not created: %v", err)
	}

	// Verify permissions (0600)
	mode := info.Mode()
	expectedMode := os.FileMode(0600)
	if mode.Perm() != expectedMode {
		t.Errorf("History file permissions = %v, want %v", mode.Perm(), expectedMode)
	}
}

func TestTrimHistoryFile(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "tfpipboy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w := NewWrapper()
	w.historyFile = filepath.Join(tmpDir, ".tfpipboy_history")

	// Create history file with more than maxHistoryLines
	f, err := os.OpenFile(w.historyFile, os.O_CREATE|os.O_WRONLY, historyFileMode)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Write maxHistoryLines + 100 lines
	for i := 0; i < maxHistoryLines+100; i++ {
		f.WriteString("test command\n")
	}
	f.Close()

	// Trim the file
	if err := w.trimHistoryFile(); err != nil {
		t.Fatalf("trimHistoryFile() error = %v", err)
	}

	// Count lines in trimmed file
	content, err := os.ReadFile(w.historyFile)
	if err != nil {
		t.Fatalf("Failed to read trimmed file: %v", err)
	}

	lines := 0
	for _, c := range content {
		if c == '\n' {
			lines++
		}
	}

	if lines > maxHistoryLines {
		t.Errorf("Trimmed file has %d lines, want <= %d", lines, maxHistoryLines)
	}
}

func TestTrimHistoryFile_NonExistent(t *testing.T) {
	w := NewWrapper()
	w.historyFile = "/nonexistent/path/.tfpipboy_history"

	// Should not error on non-existent file
	if err := w.trimHistoryFile(); err != nil {
		t.Errorf("trimHistoryFile() on non-existent file error = %v, want nil", err)
	}
}

func TestAddToHistory(t *testing.T) {
	w := NewWrapper()

	tests := []struct {
		name     string
		commands []string
		want     int // expected history length
	}{
		{
			name:     "single command",
			commands: []string{"terraform plan"},
			want:     1,
		},
		{
			name:     "multiple commands",
			commands: []string{"terraform plan", "terraform apply", "terraform destroy"},
			want:     3,
		},
		{
			name:     "duplicate consecutive",
			commands: []string{"ls", "ls", "ls"},
			want:     1, // Duplicates are not added
		},
		{
			name:     "duplicate non-consecutive",
			commands: []string{"ls", "pwd", "ls"},
			want:     3, // Non-consecutive duplicates are added
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.commandHistory = []string{} // Reset history
			w.historyIndex = 0

			for _, cmd := range tt.commands {
				w.addToHistory(cmd)
			}

			if len(w.commandHistory) != tt.want {
				t.Errorf("history length = %d, want %d", len(w.commandHistory), tt.want)
			}

			if w.historyIndex != len(w.commandHistory) {
				t.Errorf("historyIndex = %d, want %d", w.historyIndex, len(w.commandHistory))
			}
		})
	}
}

func TestHandleBuiltinCommand(t *testing.T) {
	w := NewWrapper()

	tests := []struct {
		name    string
		command string
		want    bool // true if handled
	}{
		{
			name:    "cd command",
			command: "cd /tmp",
			want:    true,
		},
		{
			name:    "help command",
			command: "help",
			want:    true,
		},
		{
			name:    "non-builtin command",
			command: "terraform plan",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.handleBuiltinCommand(tt.command)
			if got != tt.want {
				t.Errorf("handleBuiltinCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestWrapperFields(t *testing.T) {
	w := NewWrapper()

	// Test timeout default
	if w.commandTimeout <= 0 {
		t.Error("commandTimeout should be positive")
	}

	if w.commandTimeout != defaultCommandTimeout {
		t.Errorf("commandTimeout = %v, want %v", w.commandTimeout, defaultCommandTimeout)
	}

	// Test history file default
	if w.historyFile == "" {
		t.Error("historyFile should be set")
	}

	// Verify it points to user's home
	home := os.Getenv("HOME")
	if home != "" && !filepath.IsAbs(w.historyFile) {
		t.Error("historyFile should be absolute path")
	}
}

// Benchmark tests
func BenchmarkAddToHistory(b *testing.B) {
	w := NewWrapper()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.addToHistory("test command")
	}
}

func BenchmarkTrimHistoryFile(b *testing.B) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "tfpipboy-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w := NewWrapper()
	w.historyFile = filepath.Join(tmpDir, ".tfpipboy_history")

	// Create test file
	f, err := os.OpenFile(w.historyFile, os.O_CREATE|os.O_WRONLY, historyFileMode)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	for i := 0; i < maxHistoryLines+1000; i++ {
		f.WriteString("test command\n")
	}
	f.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := w.trimHistoryFile(); err != nil {
			b.Fatalf("trimHistoryFile() error = %v", err)
		}

		// Restore file for next iteration
		if i < b.N-1 {
			f, _ = os.OpenFile(w.historyFile, os.O_CREATE|os.O_WRONLY, historyFileMode)
			for j := 0; j < maxHistoryLines+1000; j++ {
				f.WriteString("test command\n")
			}
			f.Close()
		}
	}
}

// TestCommandTimeout verifies timeout is properly configured
func TestCommandTimeout(t *testing.T) {
	w := NewWrapper()

	// Default timeout should be reasonable
	if w.commandTimeout < time.Minute {
		t.Errorf("commandTimeout too short: %v", w.commandTimeout)
	}

	if w.commandTimeout > time.Hour {
		t.Errorf("commandTimeout too long: %v", w.commandTimeout)
	}

	// Should match constant
	if w.commandTimeout != defaultCommandTimeout {
		t.Errorf("commandTimeout = %v, want %v", w.commandTimeout, defaultCommandTimeout)
	}
}
