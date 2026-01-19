package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindGitRepositoryRoot(t *testing.T) {
	// Test when not in a git repository
	// Create a temporary directory that's not a git repo
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	gitRoot := FindGitRepositoryRoot()
	if gitRoot != "" {
		t.Errorf("Expected empty string when not in git repo, got: %s", gitRoot)
	}

	// Test when in a git repository
	// Initialize a git repo in temp dir
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	gitRoot = FindGitRepositoryRoot()
	if gitRoot != tempDir {
		t.Errorf("Expected git root to be %s, got: %s", tempDir, gitRoot)
	}

	// Test from a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	err = os.Chdir(subDir)
	if err != nil {
		t.Fatalf("Failed to change to subdirectory: %v", err)
	}

	gitRoot = FindGitRepositoryRoot()
	if gitRoot != tempDir {
		t.Errorf("Expected git root to be %s from subdirectory, got: %s", tempDir, gitRoot)
	}
}

func TestGetTaskDBPath(t *testing.T) {
	// Test when not in a git repository
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	dbPath := GetTaskDBPath()
	expected := "./giggum_tasks.db"
	if dbPath != expected {
		t.Errorf("Expected %s when not in git repo, got: %s", expected, dbPath)
	}

	// Test when in a git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	dbPath = GetTaskDBPath()
	expected = filepath.Join(tempDir, "giggum_tasks.db")
	if dbPath != expected {
		t.Errorf("Expected %s when in git repo, got: %s", expected, dbPath)
	}
}
