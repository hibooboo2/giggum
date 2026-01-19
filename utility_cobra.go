package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var listAgentsCmd = &cobra.Command{
	Use:   "list-agents",
	Short: "List all available agent types",
	Long:  `Display all available agent types with their descriptions.`,
	Run: func(cmd *cobra.Command, args []string) {
		listAvailableAgents()
	},
}

var showProgressCmd = &cobra.Command{
	Use:   "show-progress",
	Short: "Show agent progress for current project",
	Long:  `Display the progress and statistics of agents for the current project.`,
	Run: func(cmd *cobra.Command, args []string) {
		showAgentProgress()
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup current progress",
	Long:  `Create a backup of the current agent progress and task state.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := NewLogger("INFO", verbose, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()

		if err := backupProgress(logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error backing up progress: %v\n", err)
			os.Exit(1)
		}
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore progress from latest backup",
	Long:  `Restore the latest backup and exit.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger, err := NewLogger("INFO", verbose, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
			os.Exit(1)
		}
		defer logger.Close()

		if err := restoreProgress(logger); err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring progress: %v\n", err)
			os.Exit(1)
		}
	},
}
