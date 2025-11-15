package orchestrator

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// DefaultLogger implements the Logger interface
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
	fields map[string]interface{}
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
		fields: make(map[string]interface{}),
	}
}

// NewFileLogger creates a new logger that writes to a file
func NewFileLogger(level LogLevel, filename string) (*DefaultLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &DefaultLogger{
		level:  level,
		logger: log.New(file, "", 0),
		fields: make(map[string]interface{}),
	}, nil
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.log(LogLevelDebug, msg, args...)
	}
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.log(LogLevelInfo, msg, args...)
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.log(LogLevelWarn, msg, args...)
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.log(LogLevelError, msg, args...)
	}
}

// WithField returns a new logger with an additional field
func (l *DefaultLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &DefaultLogger{
		level:  l.level,
		logger: l.logger,
		fields: newFields,
	}
}

// WithFields returns a new logger with additional fields
func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &DefaultLogger{
		level:  l.level,
		logger: l.logger,
		fields: newFields,
	}
}

// log formats and logs a message
func (l *DefaultLogger) log(level LogLevel, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Format the message
	if len(args) > 0 {
		// Handle key-value pairs
		if len(args)%2 == 0 {
			var kvPairs []string
			for i := 0; i < len(args); i += 2 {
				key := fmt.Sprintf("%v", args[i])
				value := fmt.Sprintf("%v", args[i+1])
				kvPairs = append(kvPairs, fmt.Sprintf("%s=%s", key, value))
			}
			if len(kvPairs) > 0 {
				msg = fmt.Sprintf("%s %s", msg, strings.Join(kvPairs, " "))
			}
		} else {
			// Handle regular format string
			msg = fmt.Sprintf(msg, args...)
		}
	}

	// Add fields to the message
	if len(l.fields) > 0 {
		var fieldPairs []string
		for key, value := range l.fields {
			fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", key, value))
		}
		msg = fmt.Sprintf("%s %s", msg, strings.Join(fieldPairs, " "))
	}

	// Format the final log line
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), msg)

	l.logger.Println(logLine)
}

// SetLevel sets the logging level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current logging level
func (l *DefaultLogger) GetLevel() LogLevel {
	return l.level
}

// NopLogger is a logger that does nothing
type NopLogger struct{}

// NewNopLogger creates a new no-op logger
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// Debug does nothing
func (l *NopLogger) Debug(msg string, args ...interface{}) {}

// Info does nothing
func (l *NopLogger) Info(msg string, args ...interface{}) {}

// Warn does nothing
func (l *NopLogger) Warn(msg string, args ...interface{}) {}

// Error does nothing
func (l *NopLogger) Error(msg string, args ...interface{}) {}

// WithField returns the same logger
func (l *NopLogger) WithField(key string, value interface{}) Logger {
	return l
}

// WithFields returns the same logger
func (l *NopLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}
