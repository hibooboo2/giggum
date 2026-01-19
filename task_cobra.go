package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  `Task management commands for creating, listing, updating, and removing tasks.`,
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `Display all tasks grouped by status with their details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := handleTaskCommand("list"); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tasks: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a task interactively",
	Long:  `Create a new task by answering prompts for title, description, priority, tags, and metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := handleTaskCommand("create"); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating task: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskAddCmd = &cobra.Command{
	Use:   "add [task description]",
	Short: "Add a task from raw string",
	Long: `Add a task by providing a raw task description string.
The researcher agent will analyze and structure the task automatically.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := addTaskFromCobraArgs(taskManager, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding task: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a task interactively",
	Long:  `Remove a task by selecting it from the list of existing tasks.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := handleTaskCommand("remove"); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing task: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskSetCmd = &cobra.Command{
	Use:   "set <task_id> <property> <value>",
	Short: "Set task properties",
	Long: `Update task properties. You can update one property at a time:
  giggum task set <task_id> status completed
  giggum task set <task_id> priority high
  giggum task set <task_id> tags "frontend,urgent"

Or multiple properties at once:
  giggum task set <task_id> status=completed priority=high tags="frontend,urgent"`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := setTaskFromCobraArgs(taskManager, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting task properties: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show task statistics",
	Long:  `Display statistics about tasks grouped by status and priority.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := handleTaskCommand("stats"); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing task stats: %v\n", err)
			os.Exit(1)
		}
	},
}

var taskExecuteCmd = &cobra.Command{
	Use:   "execute <taskID>",
	Short: "Execute a task with the best suited agent",
	Long: `Execute a specific task using intelligent agent selection.
The system will analyze the task content and choose the most appropriate agent type for the job.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create logger
		logger, err := NewLogger("INFO", verbose, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()

		// Load configuration
		config, err := loadConfig(logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Override config with command line agent settings
		config.UseAgents = true
		config.AgentType = BackendDeveloper // Default fallback

		// Override timeout with command line value if provided
		if timeout > 0 {
			config.AgentTimeout = timeout
		}

		// Parse task ID
		var taskID int64
		_, err = fmt.Sscanf(args[0], "%d", &taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid task ID: %v\n", err)
			os.Exit(1)
		}

		// Get the task
		task, err := taskManager.GetTask(taskID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting task: %v\n", err)
			os.Exit(1)
		}

		if task.Status == "completed" {
			fmt.Printf("Task %d is already completed.\n", taskID)
			return
		}

		// Execute the task with the best suited agent
		err = executeTaskWithAgent(logger, config, *task, debug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing task: %v\n", err)
			os.Exit(1)
		}

		// Send webhook notification after completing task
		if webhookErr := sendWebhookNotification(logger, config, 1, true); webhookErr != nil {
			logger.Warn("Failed to send webhook notification: %v", webhookErr)
		}

		fmt.Printf("Task %d executed successfully.\n", taskID)
	},
}

func init() {
	// Add subcommands to task command
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskRemoveCmd)
	taskCmd.AddCommand(taskSetCmd)
	taskCmd.AddCommand(taskStatsCmd)
	taskCmd.AddCommand(taskExecuteCmd)
}
