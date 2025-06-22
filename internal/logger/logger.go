package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// VerboseChecker interface for checking verbose state
type VerboseChecker interface {
	IsVerbose() bool
}

// Logger provides structured logging with verbose support
type Logger struct {
	component      string
	verboseChecker VerboseChecker
	writer         io.Writer
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// New creates a new logger instance
func New(component string, verboseChecker VerboseChecker) *Logger {
	return &Logger{
		component:      component,
		verboseChecker: verboseChecker,
		writer:         os.Stderr,
	}
}

// NewWithCallback creates a new logger instance with a callback function
func NewWithCallback(component string, verboseCheck func() bool) *Logger {
	return &Logger{
		component:      component,
		verboseChecker: &callbackChecker{callback: verboseCheck},
		writer:         os.Stderr,
	}
}

// WithComponent creates a logger with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		component:      component,
		verboseChecker: l.verboseChecker,
		writer:         l.writer,
	}
}

// callbackChecker implements VerboseChecker with a callback function
type callbackChecker struct {
	callback func() bool
}

func (c *callbackChecker) IsVerbose() bool {
	if c.callback == nil {
		return false
	}
	return c.callback()
}

// Debug logs debug messages (only when verbose=true)
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.verboseChecker != nil && l.verboseChecker.IsVerbose() {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs informational messages (only when verbose=true)
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.verboseChecker != nil && l.verboseChecker.IsVerbose() {
		l.log("INFO", msg, args...)
	}
}

// Warn logs warning messages (always shown)
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log("WARN", msg, args...)
}

// Error logs error messages (always shown)
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log("ERROR", msg, args...)
}

// DebugWithFields logs debug message with structured fields
func (l *Logger) DebugWithFields(msg string, fields []Field, args ...interface{}) {
	if l.verboseChecker != nil && l.verboseChecker.IsVerbose() {
		l.logWithFields("DEBUG", msg, fields, args...)
	}
}

// InfoWithFields logs info message with structured fields
func (l *Logger) InfoWithFields(msg string, fields []Field, args ...interface{}) {
	if l.verboseChecker != nil && l.verboseChecker.IsVerbose() {
		l.logWithFields("INFO", msg, fields, args...)
	}
}

// log formats and writes log message
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.000")
	component := l.component
	if component == "" {
		component = "main"
	}

	formattedMsg := fmt.Sprintf(msg, args...)
	logLine := fmt.Sprintf("[%s] %s [%s] %s\n", timestamp, level, component, formattedMsg)

	if _, err := fmt.Fprint(l.writer, logLine); err != nil {
		// Log write failed, but we can't do much about it
		// since this is the logger itself
		_ = err
	}
}

// logWithFields formats and writes log message with structured fields
func (l *Logger) logWithFields(level, msg string, fields []Field, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.000")
	component := l.component
	if component == "" {
		component = "main"
	}

	formattedMsg := fmt.Sprintf(msg, args...)

	// Format fields
	fieldStrings := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldStrings = append(fieldStrings, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	var fieldsStr string
	if len(fieldStrings) > 0 {
		fieldsStr = fmt.Sprintf(" [%s]", strings.Join(fieldStrings, " "))
	}

	logLine := fmt.Sprintf("[%s] %s [%s] %s%s\n", timestamp, level, component, formattedMsg, fieldsStr)

	if _, err := fmt.Fprint(l.writer, logLine); err != nil {
		// Log write failed, but we can't do much about it
		// since this is the logger itself
		_ = err
	}
}

// Helper functions for common field types
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

func Count(value int) Field {
	return Field{Key: "count", Value: value}
}

func Duration(d time.Duration) Field {
	return Field{Key: "duration", Value: d}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}
