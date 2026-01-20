package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// DiscordService handles Discord bot API integration
type DiscordService struct {
	session     *discordgo.Session
	config      DiscordConfig
	logger      *Logger
	taskMgr     *TaskManager
	rateLimiter *DiscordRateLimiter
	enabled     bool
}

// DiscordRateLimiter implements rate limiting for Discord API calls
type DiscordRateLimiter struct {
	messages     []time.Time
	maxPerMinute int
	mutex        sync.Mutex
}

// NewDiscordRateLimiter creates a new rate limiter
func NewDiscordRateLimiter(maxPerMinute int) *DiscordRateLimiter {
	return &DiscordRateLimiter{
		maxPerMinute: maxPerMinute,
		messages:     make([]time.Time, 0),
	}
}

// CanSend checks if a message can be sent (rate limiting)
func (rl *DiscordRateLimiter) CanSend() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Remove messages older than 1 minute
	validMessages := make([]time.Time, 0)
	for _, msgTime := range rl.messages {
		if now.Sub(msgTime) < time.Minute {
			validMessages = append(validMessages, msgTime)
		}
	}
	rl.messages = validMessages

	// Check if we can send more messages
	return len(rl.messages) < rl.maxPerMinute
}

// RecordSent records that a message was sent
func (rl *DiscordRateLimiter) RecordSent() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.messages = append(rl.messages, time.Now())
}

// NewDiscordService creates a new Discord service instance
func NewDiscordService(config DiscordConfig, logger *Logger, taskMgr *TaskManager) (*DiscordService, error) {
	if !config.Enabled || (config.BotToken == "" && config.WebhookURL == "") {
		logger.Info("Discord service disabled - not configured")
		return &DiscordService{enabled: false, config: config, logger: logger, taskMgr: taskMgr}, nil
	}

	service := &DiscordService{
		config:  config,
		logger:  logger,
		taskMgr: taskMgr,
		enabled: true,
	}

	// Initialize rate limiter
	if config.RateLimitEnabled {
		service.rateLimiter = NewDiscordRateLimiter(config.MaxMessagesPerMinute)
	}

	// Initialize Discord bot if token is provided
	if config.BotToken != "" {
		session, err := discordgo.New("Bot " + config.BotToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create Discord session: %v", err)
		}

		session.AddHandler(service.onMessageCreate)

		service.session = session

		// Open connection
		if err := session.Open(); err != nil {
			return nil, fmt.Errorf("failed to open Discord connection: %v", err)
		}

		logger.Info("Discord bot connected successfully")
	}

	return service, nil
}

// Close closes the Discord service
func (ds *DiscordService) Close() {
	if ds.session != nil {
		ds.session.Close()
	}
}

// SendNotification sends a notification to Discord
func (ds *DiscordService) SendNotification(title, message string, iterations int, completed bool) error {
	if !ds.enabled {
		ds.logger.Debug("Discord service disabled, skipping notification")
		return nil
	}

	// Check rate limiting
	if ds.rateLimiter != nil && !ds.rateLimiter.CanSend() {
		ds.logger.Warn("Discord rate limit reached, skipping notification")
		return fmt.Errorf("rate limit reached")
	}

	// Prefer webhook URL if available, otherwise use bot
	if ds.config.WebhookURL != "" {
		return ds.sendWebhookNotification(title, message, iterations, completed)
	} else if ds.session != nil && ds.config.ChannelID != "" {
		return ds.sendBotNotification(title, message, iterations, completed)
	}

	return fmt.Errorf("no valid Discord configuration found")
}

// sendWebhookNotification sends notification via Discord webhook
func (ds *DiscordService) sendWebhookNotification(title, message string, iterations int, completed bool) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Giggum Notification",
		Description: title,
		Color:       ds.getColor(completed),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Message",
				Value:  message,
				Inline: false,
			},
			{
				Name:   "Iterations",
				Value:  strconv.Itoa(iterations),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  ds.getStatusText(completed),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	payload := map[string]interface{}{
		"embeds": []*discordgo.MessageEmbed{embed},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	resp, err := http.Post(ds.config.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	if ds.rateLimiter != nil {
		ds.rateLimiter.RecordSent()
	}

	ds.logger.Info("Discord webhook notification sent successfully")
	return nil
}

// sendBotNotification sends notification via Discord bot
func (ds *DiscordService) sendBotNotification(title, message string, iterations int, completed bool) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Giggum Notification",
		Description: title,
		Color:       ds.getColor(completed),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Message",
				Value:  message,
				Inline: false,
			},
			{
				Name:   "Iterations",
				Value:  strconv.Itoa(iterations),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  ds.getStatusText(completed),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := ds.session.ChannelMessageSendEmbed(ds.config.ChannelID, embed)
	if err != nil {
		return fmt.Errorf("failed to send bot message: %v", err)
	}

	if ds.rateLimiter != nil {
		ds.rateLimiter.RecordSent()
	}

	ds.logger.Info("Discord bot notification sent successfully")
	return nil
}

// onMessageCreate handles incoming Discord messages
func (ds *DiscordService) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if message starts with command prefix
	if !strings.HasPrefix(m.Content, ds.config.CommandPrefix) {
		return
	}

	ds.logger.Info("Received Discord command: %s", m.Content)

	// Parse command
	command := strings.TrimSpace(strings.TrimPrefix(m.Content, ds.config.CommandPrefix))
	if command == "" {
		return
	}

	// Handle different commands
	if strings.HasPrefix(command, "add ") || strings.HasPrefix(command, "create ") {
		taskStr := strings.TrimSpace(strings.TrimPrefix(command, "add "))
		if taskStr == "" {
			taskStr = strings.TrimSpace(strings.TrimPrefix(command, "create "))
		}
		if taskStr != "" {
			ds.handleTaskAdd(m, taskStr)
		}
	} else if command == "list" || command == "tasks" {
		ds.handleTaskList(m)
	} else if strings.HasPrefix(command, "status ") {
		taskID := strings.TrimSpace(strings.TrimPrefix(command, "status "))
		ds.handleTaskStatus(m, taskID)
	} else if command == "help" {
		ds.handleHelp(m)
	}
}

// handleTaskAdd handles task creation from Discord
func (ds *DiscordService) handleTaskAdd(m *discordgo.MessageCreate, taskStr string) {
	ds.logger.Info("Creating task from Discord: %s", taskStr)

	// Create task using existing task system
	if err := addTaskFromRawTaskString(ds.taskMgr, taskStr); err != nil {
		ds.sendErrorResponse(m, fmt.Sprintf("Failed to create task: %v", err))
		return
	}

	// Send confirmation
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Task Created",
		Description: fmt.Sprintf("Task successfully added to Giggum"),
		Color:       0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Task",
				Value:  taskStr,
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := ds.session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		ds.logger.Warn("Failed to send task creation confirmation: %v", err)
	}
}

// handleTaskList handles task list request from Discord
func (ds *DiscordService) handleTaskList(m *discordgo.MessageCreate) {
	tasks, err := ds.taskMgr.GetAllTasks()
	if err != nil {
		ds.sendErrorResponse(m, fmt.Sprintf("Failed to get tasks: %v", err))
		return
	}

	if len(tasks) == 0 {
		ds.sendSimpleMessage(m, "No tasks found")
		return
	}

	// Create task list embed
	var description strings.Builder
	pendingCount := 0
	inProgressCount := 0
	completedCount := 0

	for _, task := range tasks {
		if task.Status == "pending" {
			pendingCount++
		} else if task.Status == "in_progress" {
			inProgressCount++
		} else if task.Status == "completed" {
			completedCount++
		}

		statusIcon := ds.getStatusIcon(task.Status)
		priorityIcon := ds.getPriorityIcon(task.Priority)

		description.WriteString(fmt.Sprintf("%s %s [%d] %s\n",
			statusIcon, priorityIcon, task.ID, task.Title))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📋 Task List",
		Description: description.String(),
		Color:       0x0099FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Summary",
				Value:  fmt.Sprintf("Pending: %d | In Progress: %d | Completed: %d", pendingCount, inProgressCount, completedCount),
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err = ds.session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		ds.logger.Warn("Failed to send task list: %v", err)
	}
}

// handleTaskStatus handles task status request from Discord
func (ds *DiscordService) handleTaskStatus(m *discordgo.MessageCreate, taskIDStr string) {
	// Extract task ID
	taskIDStr = strings.TrimSpace(taskIDStr)
	if taskIDStr == "" {
		ds.sendErrorResponse(m, "Please provide a task ID")
		return
	}

	// Try to parse as integer
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		ds.sendErrorResponse(m, fmt.Sprintf("Invalid task ID: %s", taskIDStr))
		return
	}

	// Get task
	task, err := ds.taskMgr.GetTask(taskID)
	if err != nil {
		ds.sendErrorResponse(m, fmt.Sprintf("Task not found: %d", taskID))
		return
	}

	// Create task status embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📝 Task %d", task.ID),
		Description: task.Title,
		Color:       ds.getColorForStatus(task.Status),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  ds.getStatusIcon(task.Status) + " " + task.Status,
				Inline: true,
			},
			{
				Name:   "Priority",
				Value:  ds.getPriorityIcon(task.Priority) + " " + task.Priority,
				Inline: true,
			},
			{
				Name:   "Created",
				Value:  task.CreatedAt.Format("2006-01-02 15:04:05"),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if task.Description != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Description",
			Value:  task.Description,
			Inline: false,
		})
	}

	if task.Tags != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Tags",
			Value:  task.Tags,
			Inline: true,
		})
	}

	if task.CompletedAt != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Completed",
			Value:  task.CompletedAt.Format("2006-01-02 15:04:05"),
			Inline: true,
		})
	}

	_, err = ds.session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		ds.logger.Warn("Failed to send task status: %v", err)
	}
}

// handleHelp sends help message
func (ds *DiscordService) handleHelp(m *discordgo.MessageCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Giggum Bot Help",
		Description: "Available commands for the Giggum Discord bot",
		Color:       0x0099FF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   ds.config.CommandPrefix + "add <task>",
				Value:  "Create a new task",
				Inline: false,
			},
			{
				Name:   ds.config.CommandPrefix + "create <task>",
				Value:  "Create a new task (alias for add)",
				Inline: false,
			},
			{
				Name:   ds.config.CommandPrefix + "list",
				Value:  "Show all tasks",
				Inline: false,
			},
			{
				Name:   ds.config.CommandPrefix + "tasks",
				Value:  "Show all tasks (alias for list)",
				Inline: false,
			},
			{
				Name:   ds.config.CommandPrefix + "status <task_id>",
				Value:  "Get details of a specific task",
				Inline: false,
			},
			{
				Name:   ds.config.CommandPrefix + "help",
				Value:  "Show this help message",
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := ds.session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		ds.logger.Warn("Failed to send help message: %v", err)
	}
}

// sendErrorResponse sends an error response to Discord
func (ds *DiscordService) sendErrorResponse(m *discordgo.MessageCreate, message string) {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: message,
		Color:       0xFF0000,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := ds.session.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		ds.logger.Warn("Failed to send error response: %v", err)
	}
}

// sendSimpleMessage sends a simple message to Discord
func (ds *DiscordService) sendSimpleMessage(m *discordgo.MessageCreate, message string) {
	_, err := ds.session.ChannelMessageSend(m.ChannelID, message)
	if err != nil {
		ds.logger.Warn("Failed to send simple message: %v", err)
	}
}

// Helper functions
func (ds *DiscordService) getColor(completed bool) int {
	if completed {
		return 0x00FF00 // Green
	}
	return 0xFFA500 // Orange
}

func (ds *DiscordService) getColorForStatus(status string) int {
	switch status {
	case "completed":
		return 0x00FF00 // Green
	case "in_progress":
		return 0xFFA500 // Orange
	case "cancelled":
		return 0xFF0000 // Red
	default:
		return 0x808080 // Gray
	}
}

func (ds *DiscordService) getStatusText(completed bool) string {
	if completed {
		return "✅ Completed"
	}
	return "🔄 In Progress"
}

func (ds *DiscordService) getStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "in_progress":
		return "🔄"
	case "cancelled":
		return "❌"
	default:
		return "⏳"
	}
}

func (ds *DiscordService) getPriorityIcon(priority string) string {
	switch strings.ToLower(priority) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// ProcessDiscordResponse processes Discord responses and converts them to tasks
func ProcessDiscordResponse(logger *Logger, config Config, response string) error {
	if !config.Discord.ParseResponses || strings.TrimSpace(response) == "" {
		return nil
	}

	logger.Info("Processing Discord response: %s", response)

	// If configured to add to tasks
	if config.AddToTasks {
		if err := addToTasks(logger, response); err != nil {
			logger.Warn("Failed to add Discord response to tasks.md: %v", err)
			return err
		}
	}

	return nil
}

// ExtractTaskFromMessage attempts to extract task information from a Discord message
func ExtractTaskFromMessage(message string) (string, bool) {
	// Look for task-related patterns
	patterns := []string{
		`(?i)(?:task|todo|need to|should|implement|add|create|fix)\s+:?\s*(.+)`,
		`(?i)^\s*[-*]\s*(.+)`,
		`(?i)^\s*\d+\.\s*(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(message)
		if len(matches) > 1 {
			task := strings.TrimSpace(matches[1])
			if task != "" && len(task) > 3 { // Minimum length check
				return task, true
			}
		}
	}

	return "", false
}
