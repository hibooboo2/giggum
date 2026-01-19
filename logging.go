package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         LogLevel               `json:"level"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Method        string                 `json:"method,omitempty"`
	Path          string                 `json:"path,omitempty"`
	StatusCode    int                    `json:"status_code,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	IP            string                 `json:"ip,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Function      string                 `json:"function,omitempty"`
	File          string                 `json:"file,omitempty"`
	Line          int                    `json:"line,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Stack         string                 `json:"stack,omitempty"`
}

// StructuredLoggerInterface defines the structured logging contract
type StructuredLoggerInterface interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
	Fatal(msg string, err error, fields ...interface{})
	WithContext(ctx context.Context) StructuredLoggerInterface
	WithCorrelationID(correlationID string) StructuredLoggerInterface
	WithUserID(userID string) StructuredLoggerInterface
	WithRequestID(requestID string) StructuredLoggerInterface
	WithComponent(component string) StructuredLoggerInterface
	WithField(key string, value interface{}) StructuredLoggerInterface
	WithFields(fields map[string]interface{}) StructuredLoggerInterface
}

// StructuredLogger implements structured logging with correlation IDs
type StructuredLogger struct {
	correlationID string
	userID        string
	requestID     string
	component     string
	fields        map[string]interface{}
	output        *log.Logger
	mutex         sync.Mutex
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(component string) StructuredLoggerInterface {
	return &StructuredLogger{
		component: component,
		fields:    make(map[string]interface{}),
		output:    log.New(os.Stdout, "", 0), // No prefix, we handle formatting
	}
}

// WithContext creates a logger with context information
func (sl *StructuredLogger) WithContext(ctx context.Context) StructuredLoggerInterface {
	newLogger := sl.clone()

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		newLogger.correlationID = correlationID
	}

	if userID, ok := ctx.Value("user_id").(string); ok {
		newLogger.userID = userID
	}

	if requestID, ok := ctx.Value("request_id").(string); ok {
		newLogger.requestID = requestID
	}

	return newLogger
}

// WithCorrelationID creates a logger with correlation ID
func (sl *StructuredLogger) WithCorrelationID(correlationID string) StructuredLoggerInterface {
	newLogger := sl.clone()
	newLogger.correlationID = correlationID
	return newLogger
}

// WithUserID creates a logger with user ID
func (sl *StructuredLogger) WithUserID(userID string) StructuredLoggerInterface {
	newLogger := sl.clone()
	newLogger.userID = userID
	return newLogger
}

// WithRequestID creates a logger with request ID
func (sl *StructuredLogger) WithRequestID(requestID string) StructuredLoggerInterface {
	newLogger := sl.clone()
	newLogger.requestID = requestID
	return newLogger
}

// WithComponent creates a logger with component name
func (sl *StructuredLogger) WithComponent(component string) StructuredLoggerInterface {
	newLogger := sl.clone()
	newLogger.component = component
	return newLogger
}

// WithField creates a logger with an additional field
func (sl *StructuredLogger) WithField(key string, value interface{}) StructuredLoggerInterface {
	newLogger := sl.clone()
	newLogger.fields[key] = value
	return newLogger
}

// WithFields creates a logger with additional fields
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) StructuredLoggerInterface {
	newLogger := sl.clone()
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(msg string, fields ...interface{}) {
	sl.log(LevelDebug, msg, nil, fields...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(msg string, fields ...interface{}) {
	sl.log(LevelInfo, msg, nil, fields...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(msg string, fields ...interface{}) {
	sl.log(LevelWarn, msg, nil, fields...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(msg string, err error, fields ...interface{}) {
	sl.log(LevelError, msg, err, fields...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(msg string, err error, fields ...interface{}) {
	sl.log(LevelFatal, msg, err, fields...)
	os.Exit(1)
}

// log is the core logging method
func (sl *StructuredLogger) log(level LogLevel, msg string, err error, fields ...interface{}) {
	entry := LogEntry{
		Timestamp:     time.Now().UTC(),
		Level:         level,
		Message:       msg,
		CorrelationID: sl.correlationID,
		UserID:        sl.userID,
		RequestID:     sl.requestID,
		Component:     sl.component,
		Fields:        sl.fields,
	}

	// Add error information if provided
	if err != nil {
		entry.Error = err.Error()
		// Add stack trace for error levels
		if level >= LevelError {
			entry.Stack = getStackTrace()
		}
	}

	// Add caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		entry.Function = fn.Name()
		entry.File = file
		entry.Line = line
	}

	// Add additional fields
	if len(fields) > 0 {
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}

		// Handle key-value pairs
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				key := fmt.Sprintf("%v", fields[i])
				value := fields[i+1]
				entry.Fields[key] = value
			}
		}
	}

	// Output the log entry
	sl.outputEntry(entry)
}

// outputEntry formats and outputs the log entry
func (sl *StructuredLogger) outputEntry(entry LogEntry) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	// For production, you might want to use a JSON formatter
	// For now, we'll use a readable text format
	var builder strings.Builder

	// Timestamp and level
	builder.WriteString(fmt.Sprintf("[%s] %s",
		entry.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		entry.Level.String()))

	// Component
	if entry.Component != "" {
		builder.WriteString(fmt.Sprintf(" [%s]", entry.Component))
	}

	// Correlation ID
	if entry.CorrelationID != "" {
		builder.WriteString(fmt.Sprintf(" [cid:%s]", entry.CorrelationID))
	}

	// User ID
	if entry.UserID != "" {
		builder.WriteString(fmt.Sprintf(" [uid:%s]", entry.UserID))
	}

	// Request ID
	if entry.RequestID != "" {
		builder.WriteString(fmt.Sprintf(" [rid:%s]", entry.RequestID))
	}

	// HTTP information
	if entry.Method != "" {
		builder.WriteString(fmt.Sprintf(" [%s %s]", entry.Method, entry.Path))
		if entry.StatusCode > 0 {
			builder.WriteString(fmt.Sprintf(" %d", entry.StatusCode))
		}
		if entry.Duration > 0 {
			builder.WriteString(fmt.Sprintf(" (%v)", entry.Duration))
		}
	}

	// Main message
	builder.WriteString(fmt.Sprintf(" - %s", entry.Message))

	// Error
	if entry.Error != "" {
		builder.WriteString(fmt.Sprintf(" - Error: %s", entry.Error))
	}

	// Function and location
	if entry.Function != "" {
		builder.WriteString(fmt.Sprintf(" - %s (%s:%d)", entry.Function, entry.File, entry.Line))
	}

	// Additional fields
	if len(entry.Fields) > 0 {
		builder.WriteString(" - Fields: ")
		first := true
		for k, v := range entry.Fields {
			if !first {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
	}

	// Stack trace (for errors)
	if entry.Stack != "" {
		builder.WriteString(fmt.Sprintf("\nStack Trace:\n%s", entry.Stack))
	}

	// Output
	sl.output.Println(builder.String())

	// If fatal, we already handled exit in Fatal method
}

// clone creates a copy of the logger
func (sl *StructuredLogger) clone() *StructuredLogger {
	newLogger := &StructuredLogger{
		correlationID: sl.correlationID,
		userID:        sl.userID,
		requestID:     sl.requestID,
		component:     sl.component,
		fields:        make(map[string]interface{}),
		output:        sl.output,
	}

	// Copy fields
	for k, v := range sl.fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// getStackTrace captures the current stack trace
func getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// RequestLoggingMiddleware creates middleware for HTTP request logging
func RequestLoggingMiddleware(logger StructuredLoggerInterface) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Don't log health checks
		if param.Path == "/health" {
			return ""
		}

		// Create correlation ID if not present
		correlationID := ""
		if cid, ok := param.Keys["correlation_id"].(string); ok {
			correlationID = cid
		}
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Create log entry
		entry := LogEntry{
			Timestamp:     param.TimeStamp,
			Level:         LevelInfo,
			Message:       "HTTP Request",
			CorrelationID: correlationID,
			Method:        param.Method,
			Path:          param.Path,
			StatusCode:    param.StatusCode,
			Duration:      param.Latency,
			IP:            param.ClientIP,
			UserAgent:     param.Request.UserAgent(),
			Component:     "http",
		}

		// Add user ID if available
		if userID, exists := param.Keys["user_id"]; exists {
			if uid, ok := userID.(string); ok {
				entry.UserID = uid
			}
		}

		// Add error information if request failed
		if param.ErrorMessage != "" {
			entry.Level = LevelError
			entry.Error = param.ErrorMessage
		}

		// Format output (using text format for simplicity)
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("[%s] %s",
			entry.Timestamp.Format("2006-01-02T15:04:05.000Z"),
			entry.Level.String()))

		if entry.Component != "" {
			builder.WriteString(fmt.Sprintf(" [%s]", entry.Component))
		}

		if entry.CorrelationID != "" {
			builder.WriteString(fmt.Sprintf(" [cid:%s]", entry.CorrelationID))
		}

		if entry.UserID != "" {
			builder.WriteString(fmt.Sprintf(" [uid:%s]", entry.UserID))
		}

		if entry.RequestID != "" {
			builder.WriteString(fmt.Sprintf(" [rid:%s]", entry.RequestID))
		}

		builder.WriteString(fmt.Sprintf(" - %s - [%s %s] %d (%v)",
			entry.Message, entry.Method, entry.Path, entry.StatusCode, entry.Duration))

		if entry.IP != "" {
			builder.WriteString(fmt.Sprintf(" - IP: %s", entry.IP))
		}

		if entry.Error != "" {
			builder.WriteString(fmt.Sprintf(" - Error: %s", entry.Error))
		}

		builder.WriteString("\n")
		return builder.String()
	})
}

// CorrelationIDMiddleware adds correlation ID to request context
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get correlation ID from header
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			// Generate new correlation ID
			correlationID = uuid.New().String()
		}

		// Add to context
		c.Set("correlation_id", correlationID)

		// Add header to response
		c.Header("X-Correlation-ID", correlationID)

		// Also set in gin.Keys for logger middleware
		c.Keys["correlation_id"] = correlationID

		c.Next()
	}
}

// RequestIDMiddleware adds unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Keys["request_id"] = requestID
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Global logger instance
var globalLogger StructuredLoggerInterface

// InitLogger initializes the global logger
func InitLogger(component string) {
	globalLogger = NewStructuredLogger(component)
}

// GetLogger returns the global logger instance
func GetLogger() StructuredLoggerInterface {
	if globalLogger == nil {
		InitLogger("giggum")
	}
	return globalLogger
}

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...interface{}) {
	GetLogger().Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...interface{}) {
	GetLogger().Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...interface{}) {
	GetLogger().Warn(msg, fields...)
}

// Error logs an error message using the global logger
func Error(msg string, err error, fields ...interface{}) {
	GetLogger().Error(msg, err, fields...)
}

// Fatal logs a fatal message using the global logger
func Fatal(msg string, err error, fields ...interface{}) {
	GetLogger().Fatal(msg, err, fields...)
}

// WithContext creates a logger from context using the global logger
func WithContext(ctx context.Context) StructuredLoggerInterface {
	return GetLogger().WithContext(ctx)
}

// WithCorrelationID creates a logger with correlation ID using the global logger
func WithCorrelationID(correlationID string) StructuredLoggerInterface {
	return GetLogger().WithCorrelationID(correlationID)
}

// WithUserID creates a logger with user ID using the global logger
func WithUserID(userID string) StructuredLoggerInterface {
	return GetLogger().WithUserID(userID)
}

// WithComponent creates a logger with component name using the global logger
func WithComponent(component string) StructuredLoggerInterface {
	return GetLogger().WithComponent(component)
}
