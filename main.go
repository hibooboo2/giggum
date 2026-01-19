package main

import (
	"fmt"
	"os"
)

func main() {
	// Initialize task manager for use by commands
	dbPath := GetTaskDBPath()
	var err error
	taskManager, err = NewTaskManager(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create task manager: %v", err)
	}
	defer taskManager.Close()

	// Execute Cobra CLI
	execute()
}
