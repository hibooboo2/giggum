package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GiggumError represents a structured error type for the giggum system
type GiggumError struct {
	Code    string
	Message string
	Cause   error
}

func (e *GiggumError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GiggumError) Unwrap() error {
	return e.Cause
}

// Error codes
const (
	ErrCodeDatabase      = "DATABASE_ERROR"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeAgent         = "AGENT_ERROR"
	ErrCodeWebhook       = "WEBHOOK_ERROR"
	ErrCodeFileOperation = "FILE_ERROR"
	ErrCodeConfiguration = "CONFIG_ERROR"
)

// validateWebhookURL validates webhook URL format
func validateWebhookURL(webhookURL string) error {
	if webhookURL == "" {
		return nil // Empty URL is valid (webhook disabled)
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return &GiggumError{
			Code:    ErrCodeValidation,
			Message: "Invalid webhook URL format",
			Cause:   err,
		}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return &GiggumError{
			Code:    ErrCodeValidation,
			Message: "Webhook URL must use http or https scheme",
		}
	}

	if parsedURL.Host == "" {
		return &GiggumError{
			Code:    ErrCodeValidation,
			Message: "Webhook URL must have a host",
		}
	}

	return nil
}

// validateAgentType validates agent type configuration
func validateAgentType(config *Config) error {
	// Always require an agent type since we always use agents
	if config.AgentType == "" {
		return &GiggumError{
			Code:    ErrCodeConfiguration,
			Message: "Agent type is required",
		}
	}

	// Check if agent type is valid
	_, err := GetAgentPrompt(config.AgentType)
	if err != nil {
		return &GiggumError{
			Code:    ErrCodeConfiguration,
			Message: fmt.Sprintf("Invalid agent type: %s", config.AgentType),
			Cause:   err,
		}
	}

	return nil
}

// sanitizeWebhookResponse sanitizes webhook response content
func sanitizeWebhookResponse(response string) string {
	if response == "" {
		return ""
	}

	// Remove potential HTML tags and their content (basic sanitization)
	sanitized := response
	sanitized = removeHTMLContent(sanitized, "script")
	sanitized = removeHTMLContent(sanitized, "iframe")

	// Trim whitespace and limit length
	sanitized = strings.TrimSpace(sanitized)
	if len(sanitized) > 10000 { // Limit to 10KB
		sanitized = sanitized[:10000]
	}

	return sanitized
}

// removeHTMLContent removes HTML tags and their content
func removeHTMLContent(s, tag string) string {
	startTag := "<" + tag
	endTag := "</" + tag + ">"

	for {
		startIdx := strings.Index(strings.ToLower(s), startTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(strings.ToLower(s[startIdx:]), endTag)
		if endIdx == -1 {
			break
		}

		endIdx += startIdx + len(endTag)
		s = s[:startIdx] + s[endIdx:]
	}

	return s
}

// validateConfig performs comprehensive configuration validation
func validateConfig(config *Config) error {
	if config == nil {
		return &GiggumError{
			Code:    ErrCodeConfiguration,
			Message: "Configuration cannot be nil",
		}
	}

	// Validate webhook URL
	if err := validateWebhookURL(config.WebhookURL); err != nil {
		return fmt.Errorf("webhook URL validation failed: %w", err)
	}

	// Validate agent configuration
	if err := validateAgentType(config); err != nil {
		return fmt.Errorf("agent configuration validation failed: %w", err)
	}

	return nil
}

// safeFileOperation performs file operations with error handling
func safeFileOperation(operation string, filePath string, fileOperation func() error) error {
	if filePath == "" {
		return &GiggumError{
			Code:    ErrCodeValidation,
			Message: "File path cannot be empty",
		}
	}

	err := fileOperation()
	if err != nil {
		return &GiggumError{
			Code:    ErrCodeFileOperation,
			Message: fmt.Sprintf("Failed to %s file %s", operation, filePath),
			Cause:   err,
		}
	}

	return nil
}

// createTimeout creates a timeout for operations
func createTimeout(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

// isTimeoutError checks if an error is a timeout
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "deadline exceeded")
}
