package main

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestDiscordRateLimiter(t *testing.T) {
	// Test rate limiter with 3 messages per minute
	rl := NewDiscordRateLimiter(3)

	// Should be able to send initially
	if !rl.CanSend() {
		t.Error("Expected to be able to send initially")
	}

	// Record first message
	rl.RecordSent()
	if !rl.CanSend() {
		t.Error("Expected to be able to send after first message")
	}

	// Record second message
	rl.RecordSent()
	if !rl.CanSend() {
		t.Error("Expected to be able to send after second message")
	}

	// Record third message
	rl.RecordSent()
	if rl.CanSend() {
		t.Error("Expected to hit rate limit after third message")
	}
}

func TestExtractTaskFromMessage(t *testing.T) {
	tests := []struct {
		message      string
		expectedTask string
		shouldMatch  bool
	}{
		{"Task: Implement user authentication", "user authentication", true},
		{"TODO: Fix the login bug", "the login bug", true},
		{"Need to add API endpoint", "add API endpoint", true},
		{"Should create a new component", "create a new component", true},
		{"- Implement dark mode", "dark mode", true},
		{"* Fix the responsive layout", "the responsive layout", true},
		{"1. Add validation to forms", "validation to forms", true},
		{"Just a random message", "", false},
		{"hi", "", false}, // Too short
	}

	for _, test := range tests {
		task, matched := ExtractTaskFromMessage(test.message)
		if matched != test.shouldMatch {
			t.Errorf("Expected match=%v for message '%s', got %v", test.shouldMatch, test.message, matched)
		}
		if matched && task != test.expectedTask {
			t.Errorf("Expected task '%s' for message '%s', got '%s'", test.expectedTask, test.message, task)
		}
	}
}

func TestDiscordServiceDisabled(t *testing.T) {
	logger, _ := NewLogger("DEBUG", false, "")
	defer logger.Close()

	config := DiscordConfig{
		Enabled: false,
	}

	service, err := NewDiscordService(config, logger, nil)
	if err != nil {
		t.Fatalf("Expected no error creating disabled Discord service, got %v", err)
	}

	if service.enabled {
		t.Error("Expected service to be disabled")
	}

	// Should not error when trying to send notification
	err = service.SendNotification("Test", "Test message", 1, false)
	if err != nil {
		t.Errorf("Expected no error when sending notification with disabled service, got %v", err)
	}
}

func TestDiscordServiceConfigValidation(t *testing.T) {
	logger, _ := NewLogger("DEBUG", false, "")
	defer logger.Close()

	// Test with enabled but no token or webhook
	config := DiscordConfig{
		Enabled: true,
	}

	service, err := NewDiscordService(config, logger, nil)
	if err != nil {
		t.Fatalf("Expected no error creating Discord service with invalid config, got %v", err)
	}

	if service.enabled {
		t.Error("Expected service to be disabled when no valid config provided")
	}
}

func TestProcessDiscordResponse(t *testing.T) {
	logger, _ := NewLogger("DEBUG", false, "")
	defer logger.Close()

	// Test with parsing disabled
	config := Config{
		Discord: DiscordConfig{
			ParseResponses: false,
		},
		AddToTasks: false,
	}

	err := ProcessDiscordResponse(logger, config, "Test response")
	if err != nil {
		t.Errorf("Expected no error when parsing is disabled, got %v", err)
	}

	// Test with empty response
	config.Discord.ParseResponses = true
	err = ProcessDiscordResponse(logger, config, "")
	if err != nil {
		t.Errorf("Expected no error with empty response, got %v", err)
	}

	// Test with empty whitespace response
	err = ProcessDiscordResponse(logger, config, "   \n\t  ")
	if err != nil {
		t.Errorf("Expected no error with whitespace-only response, got %v", err)
	}
}

func TestDiscordServiceColors(t *testing.T) {
	logger, _ := NewLogger("DEBUG", false, "")
	defer logger.Close()

	config := DiscordConfig{
		Enabled: true,
	}

	service := &DiscordService{
		config: config,
		logger: logger,
	}

	// Test colors for completed/incompleted
	if service.getColor(true) != 0x00FF00 {
		t.Errorf("Expected green color for completed tasks, got %x", service.getColor(true))
	}

	if service.getColor(false) != 0xFFA500 {
		t.Errorf("Expected orange color for incomplete tasks, got %x", service.getColor(false))
	}

	// Test status colors
	statusColors := map[string]int{
		"completed":   0x00FF00,
		"in_progress": 0xFFA500,
		"cancelled":   0xFF0000,
		"pending":     0x808080,
		"unknown":     0x808080,
	}

	for status, expectedColor := range statusColors {
		color := service.getColorForStatus(status)
		if color != expectedColor {
			t.Errorf("Expected color %x for status '%s', got %x", expectedColor, status, color)
		}
	}
}

func TestDiscordServiceIcons(t *testing.T) {
	logger, _ := NewLogger("DEBUG", false, "")
	defer logger.Close()

	service := &DiscordService{
		logger: logger,
	}

	// Test status icons
	statusIcons := map[string]string{
		"completed":   "✅",
		"in_progress": "🔄",
		"cancelled":   "❌",
		"pending":     "⏳",
		"unknown":     "⏳",
	}

	for status, expectedIcon := range statusIcons {
		icon := service.getStatusIcon(status)
		if icon != expectedIcon {
			t.Errorf("Expected icon '%s' for status '%s', got '%s'", expectedIcon, status, icon)
		}
	}

	// Test priority icons
	priorityIcons := map[string]string{
		"high":    "🔴",
		"medium":  "🟡",
		"low":     "🟢",
		"unknown": "⚪",
	}

	for priority, expectedIcon := range priorityIcons {
		icon := service.getPriorityIcon(priority)
		if icon != expectedIcon {
			t.Errorf("Expected icon '%s' for priority '%s', got '%s'", expectedIcon, priority, icon)
		}
	}
}

func TestDiscordRateLimiterTimeWindow(t *testing.T) {
	rl := NewDiscordRateLimiter(2)

	// Send two messages
	rl.RecordSent()
	rl.RecordSent()

	// Should hit limit
	if rl.CanSend() {
		t.Error("Expected to hit rate limit after 2 messages")
	}

	// Simulate time passing (remove old messages)
	rl.mutex.Lock()
	rl.messages = []time.Time{time.Now().Add(-61 * time.Second)} // Message older than 1 minute
	rl.mutex.Unlock()

	// Should be able to send again
	if !rl.CanSend() {
		t.Error("Expected to be able to send after time window passed")
	}
}

// Mock Discord session for testing
type mockDiscordSession struct {
	messages []mockMessage
}

type mockMessage struct {
	channelID string
	content   string
	embed     *discordgo.MessageEmbed
}

func (m *mockDiscordSession) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	m.messages = append(m.messages, mockMessage{
		channelID: channelID,
		content:   content,
	})
	return &discordgo.Message{}, nil
}

func (m *mockDiscordSession) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	m.messages = append(m.messages, mockMessage{
		channelID: channelID,
		embed:     embed,
	})
	return &discordgo.Message{}, nil
}
