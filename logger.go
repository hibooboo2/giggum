package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	verbose bool
}

// NewLogger creates a new logger instance using slog
func NewLogger(logLevel string, verbose bool, logFilePath string) (*Logger, error) {
	// Parse log level
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if logFilePath != "" {
		// Create file handler
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file '%s': %v", logFilePath, err)
		}
		handler = slog.NewTextHandler(file, opts)
	} else {
		// Use stdout handler
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		Logger:  logger,
		verbose: verbose,
	}, nil
}

// Close closes any resources (placeholder for compatibility)
func (l *Logger) Close() error {
	// slog handles resources automatically, no explicit close needed
	return nil
}

// Debug logs a debug message with custom verbose handling
func (l *Logger) Debug(msg string, args ...any) {
	if l.verbose {
		l.Logger.Debug(msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}
