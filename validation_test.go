package main

import (
	"errors"
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		expectErr bool
		errCode   string
	}{
		{
			name:      "Empty URL should be valid",
			url:       "",
			expectErr: false,
		},
		{
			name:      "Valid HTTPS URL",
			url:       "https://example.com/webhook",
			expectErr: false,
		},
		{
			name:      "Valid HTTP URL",
			url:       "http://localhost:8080/webhook",
			expectErr: false,
		},
		{
			name:      "Invalid URL format",
			url:       "not-a-url",
			expectErr: true,
			errCode:   ErrCodeValidation,
		},
		{
			name:      "Invalid scheme",
			url:       "ftp://example.com/webhook",
			expectErr: true,
			errCode:   ErrCodeValidation,
		},
		{
			name:      "Missing host",
			url:       "https:///webhook",
			expectErr: true,
			errCode:   ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				var giggumErr *GiggumError
				if !errors.As(err, &giggumErr) {
					t.Errorf("Expected GiggumError but got %T", err)
					return
				}

				if giggumErr.Code != tt.errCode {
					t.Errorf("Expected error code %s but got %s", tt.errCode, giggumErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateAgentType(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
		errCode   string
	}{
		{
			name: "Valid config with agents",
			config: &Config{
				UseAgents: true,
				AgentType: Tester,
			},
			expectErr: false,
		},
		{
			name: "Valid config with default agent",
			config: &Config{
				UseAgents: true,
				AgentType: BackendDeveloper,
			},
			expectErr: false,
		},
		{
			name: "Missing agent type when agents enabled",
			config: &Config{
				UseAgents: true,
				AgentType: "",
			},
			expectErr: true,
			errCode:   ErrCodeConfiguration,
		},
		{
			name: "Invalid agent type",
			config: &Config{
				UseAgents: true,
				AgentType: AgentType("invalid"),
			},
			expectErr: true,
			errCode:   ErrCodeConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentType(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				var giggumErr *GiggumError
				if !errors.As(err, &giggumErr) {
					t.Errorf("Expected GiggumError but got %T", err)
					return
				}

				if giggumErr.Code != tt.errCode {
					t.Errorf("Expected error code %s but got %s", tt.errCode, giggumErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSanitizeWebhookResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Normal text",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "Text with script tags",
			input:    "Hello <script>alert('xss')</script> world!",
			expected: "Hello  world!",
		},
		{
			name:     "Text with iframe tags",
			input:    "Hello <iframe>content</iframe> world!",
			expected: "Hello  world!",
		},
		{
			name:     "Whitespace trimming",
			input:    "   Hello, world!   ",
			expected: "Hello, world!",
		},
		{
			name:     "Long text truncation",
			input:    string(make([]byte, 15000)), // 15KB
			expected: string(make([]byte, 10000)), // Should be truncated to 10KB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeWebhookResponse(tt.input)
			if len(result) > 10000 {
				t.Errorf("Result length %d exceeds maximum 10000", len(result))
			}

			// For truncation test, check length
			if tt.name == "Long text truncation" {
				if len(result) != 10000 {
					t.Errorf("Expected truncated length 10000, got %d", len(result))
				}
			} else if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGiggumError(t *testing.T) {
	// Test error without cause
	err1 := &GiggumError{
		Code:    ErrCodeValidation,
		Message: "Test error",
	}

	expected1 := "[VALIDATION_ERROR] Test error"
	if err1.Error() != expected1 {
		t.Errorf("Expected %q, got %q", expected1, err1.Error())
	}

	// Test error with cause
	cause := &GiggumError{
		Code:    ErrCodeDatabase,
		Message: "Database error",
	}

	err2 := &GiggumError{
		Code:    ErrCodeValidation,
		Message: "Test error",
		Cause:   cause,
	}

	expected2 := "[VALIDATION_ERROR] Test error: [DATABASE_ERROR] Database error"
	if err2.Error() != expected2 {
		t.Errorf("Expected %q, got %q", expected2, err2.Error())
	}

	// Test Unwrap
	if err2.Unwrap() != cause {
		t.Errorf("Unwrap should return the cause error")
	}
}
