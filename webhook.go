package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// processWebhookResponse processes a webhook response by sending it to opencode to create a task list
func processWebhookResponse(logger *Logger, config Config, response string) error {
	if strings.TrimSpace(response) == "" {
		logger.Debug("Empty webhook response, skipping processing")
		return nil
	}

	logger.Info("Processing webhook response: %s", response)

	// If configured to add to tasks.md
	if config.AddToTasks {
		if err := addToTasks(logger, response); err != nil {
			logger.Warn("Failed to add webhook response to tasks.md: %v", err)
			return err
		}
	}

	return nil
}

// addToTasks adds a response to the tasks.md file using opencode for processing
func addToTasks(logger *Logger, response string) error {
	// If response is empty, skip processing
	if strings.TrimSpace(response) == "" {
		logger.Debug("Empty response, skipping task addition")
		return nil
	}

	// Use opencode to process the webhook response into a formatted task list
	logger.Info("Processing webhook response with opencode to create task list...")

	// Create opencode prompt for task processing
	opencodePrompt := fmt.Sprintf(`Process the following webhook response and extract/create a proper task list. 
The response may contain new tasks, feedback, or requirements. Convert them into a clean, organized task list format.

Webhook Response:
%s

Requirements:
1. Extract individual tasks from the response
2. Format each task as a markdown list item starting with "- "
3. Make tasks specific and actionable
4. Remove any duplicate or irrelevant content
5. If no clear tasks are found, respond with "No tasks found"
6. Output ONLY the task list, no explanations

Example output format:
- Implement user authentication system
- Add API endpoint for user registration
- Create login form component
`, response)

	// Build the opencode command
	args := []string{"run", "--model", "opencode/big-pickle"}
	cmd := exec.Command("opencode", args...)
	cmd.Env = append(os.Environ(), "OPENAI_BASE_URL=http://100.83.162.29:1234")

	// Provide the prompt via stdin
	cmd.Stdin = strings.NewReader(opencodePrompt)

	// Capture the output
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	// Run the command
	err := cmd.Run()
	if err != nil {
		logger.Warn("Failed to process webhook response with opencode: %v", err)
		logger.Warn("Using fallback: adding raw response as task")
		return addRawTask(logger, response)
	}

	// Get the processed task list
	processedTasks := strings.TrimSpace(output.String())

	// Check if opencode found any tasks
	if processedTasks == "" || strings.Contains(strings.ToLower(processedTasks), "no tasks found") {
		logger.Info("No tasks found in webhook response")
		return nil
	}

	// Read current tasks.md content
	content, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	// Create new content with the processed tasks added
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	// Add a separator and the new tasks
	newContent += "\n# Tasks from Webhook Response\n"
	newContent += processedTasks + "\n"

	// Write back to tasks.md
	if err := os.WriteFile("tasks.md", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to tasks.md: %v", err)
	}

	logger.Info("Added processed tasks to tasks.md from webhook response")
	return nil
}

// addRawTask is a fallback function that adds the raw response as a single task
func addRawTask(logger *Logger, response string) error {
	// Read current tasks.md content
	content, err := os.ReadFile("tasks.md")
	if err != nil {
		return fmt.Errorf("failed to read tasks.md: %v", err)
	}

	// Create new content with the response added
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("- %s\n", response)

	// Write back to tasks.md
	if err := os.WriteFile("tasks.md", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write to tasks.md: %v", err)
	}

	logger.Info("Added raw response to tasks.md: %s", response)
	return nil
}

// sendWebhookNotification sends a notification to the configured webhook URL
func sendWebhookNotification(logger *Logger, config Config, iterations int, completed bool) error {
	if config.WebhookURL == "" {
		logger.Debug("No webhook URL configured, skipping notification")
		return nil
	}

	// Prepare webhook payload
	payload := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"iterations":     iterations,
		"completed":      completed,
		"message":        "Ralph Wiggum execution completed",
		"reply_endpoint": "http://localhost:8080/reply",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	// Send HTTP POST request
	resp, err := http.Post(config.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	logger.Info("Webhook notification sent successfully to %s", config.WebhookURL)

	// If ParseReply is enabled, wait for and process the webhook response
	if config.ParseReply {
		// Read the immediate response from the webhook
		respdata, _ := io.ReadAll(resp.Body)
		response := strings.TrimSpace(string(respdata))

		if response != "" {
			if err := processWebhookResponse(logger, config, response); err != nil {
				logger.Warn("Failed to process webhook response: %v", err)
			}
		} else {
			logger.Debug("No immediate response received from webhook")
		}
	}
	return nil
}
