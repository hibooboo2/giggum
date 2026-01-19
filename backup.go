package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// backupProgress creates a timestamped backup of progress.txt
func backupProgress(logger *Logger) error {
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("progress_%s.txt", timestamp))

	// Read the original file
	content, err := os.ReadFile("progress.txt")
	if err != nil {
		return fmt.Errorf("failed to read progress.txt: %v", err)
	}

	// Write to backup file
	if err := os.WriteFile(backupFile, content, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	logger.Debug("Created backup: %s", backupFile)

	return nil
}

// restoreProgress restores progress.txt from the latest backup
func restoreProgress(logger *Logger) error {
	backupDir := "backups"
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup directory '%s' does not exist", backupDir)
	}

	// Find the most recent backup file
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if strings.HasPrefix(name, "progress_") && strings.HasSuffix(name, ".txt") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = filepath.Join(backupDir, name)
			}
		}
	}

	if latestFile == "" {
		return fmt.Errorf("no backup files found in %s", backupDir)
	}

	// Read backup file
	content, err := os.ReadFile(latestFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %v", err)
	}

	// Write to progress.txt
	if err := os.WriteFile("progress.txt", content, 0644); err != nil {
		return fmt.Errorf("failed to restore progress.txt: %v", err)
	}

	logger.Info("Restored progress.txt from: %s", latestFile)

	return nil
}
