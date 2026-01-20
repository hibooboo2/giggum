package main

import (
	"fmt"
	"os"
)

func runIterations(logger *Logger, config Config, iterations int, debug bool, discordService *DiscordService) {
	// Agent-only execution - validate agent type exists
	if config.AgentType == "" {
		config.AgentType = BackendDeveloper
	}

	for i := 1; i <= iterations; i++ {
		var err error
		fmt.Printf("Running iteration %d/%d with %s agent...\n", i, iterations, config.AgentType)

		// Execute using agent only - no traditional execution path
		// err := runAgentWithOutputCapture(logger, config.AgentType, config.PromptCommand, debug)
		if err != nil {
			if err.Error() == "COMPLETED" {
				// Send webhook notification for early completion
				if webhookErr := sendWebhookNotification(logger, config, i, true); webhookErr != nil {
					logger.Warn("Failed to send webhook notification: %v", webhookErr)
				}
				// Send Discord notification for early completion
				if discordErr := sendDiscordNotification(logger, config, i, true, discordService); discordErr != nil {
					logger.Warn("Failed to send Discord notification: %v", discordErr)
				}
				return // Exit early since tasks are complete
			}

			// Enhanced error handling for agent-only execution
			logger.Error("Agent execution failed in iteration %d: %v", i, err)
			// logger.Error("Agent type: %s, Task: %s", config.AgentType, config.PromptCommand)

			fmt.Fprintf(os.Stderr, "Error in iteration %d with %s agent: %v\n", i, config.AgentType, err)
			fmt.Fprintf(os.Stderr, "Note: This system uses agent-only execution - no fallback mode available.\n")

			if i == iterations {
				logger.Error("Final iteration failed, terminating")
				os.Exit(1)
			}
			logger.Info("Continuing to next iteration...")
			continue
		}
	}
}
