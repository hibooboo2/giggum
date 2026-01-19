package main

import "strings"

// Status display mappings
var statusIcons = map[string]string{
	"pending":     "⏳",
	"in_progress": "🔄",
	"completed":   "✅",
	"cancelled":   "❌",
}

var statusNames = map[string]string{
	"pending":     "Pending",
	"in_progress": "In Progress",
	"completed":   "Completed",
	"cancelled":   "Cancelled",
}

func getPriorityIcon(priority string) string {
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

func getPriorityNum(priority string) int {
	switch strings.ToLower(priority) {
	case "high":
		return 10
	case "medium":
		return 5
	case "low":
		return 1
	default:
		return 0
	}
}
