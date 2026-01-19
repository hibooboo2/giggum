package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WebServer represents the HTTP server for giggum PWA integration
type WebServer struct {
	port int
}

// NewWebServer creates a new web server instance
func NewWebServer(port int) *WebServer {
	return &WebServer{port: port}
}

// AgentResponse represents the response structure for agent information
type AgentResponse struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Start starts the web server
func (ws *WebServer) Start() error {
	// Set Gin mode to release for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// Add CORS middleware for PWA
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Serve static files for PWA
	router.Static("/static", "./web/static")
	router.StaticFile("/", "./web/index.html")
	router.StaticFile("/manifest.json", "./web/manifest.json")
	router.StaticFile("/service-worker.js", "./web/service-worker.js")

	// API routes
	api := router.Group("/api")
	{
		api.GET("/agents", ws.listAgents)
		api.GET("/agents/:type", ws.getAgent)
		api.POST("/notify", ws.sendNotification)
		api.GET("/notifications", ws.getNotifications)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "giggum-web",
		})
	})

	// Start server
	addr := fmt.Sprintf(":%d", ws.port)
	return router.Run(addr)
}

// listAgents returns all available agent types
func (ws *WebServer) listAgents(c *gin.Context) {
	agents := GetAgentPrompts()
	var response []AgentResponse

	for _, agent := range agents {
		response = append(response, AgentResponse{
			Type:        string(agent.Type),
			Name:        agent.Name,
			Description: agent.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": response,
		"count":  len(response),
	})
}

// getAgent returns details for a specific agent type
func (ws *WebServer) getAgent(c *gin.Context) {
	agentType := c.Param("type")

	// Validate agent type
	validTypes := ListAgentTypes()
	found := false
	for _, validType := range validTypes {
		if string(validType) == agentType {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error":           fmt.Sprintf("Agent type '%s' not found", agentType),
			"available_types": validTypes,
		})
		return
	}

	agent, err := GetAgentPrompt(AgentType(agentType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get agent: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"type":          string(agent.Type),
		"name":          agent.Name,
		"description":   agent.Description,
		"system_prompt": agent.SystemPrompt,
		"task_prompt":   agent.TaskPrompt,
	})
}

// NotificationMessage represents a notification payload
type NotificationMessage struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Agent   string `json:"agent,omitempty"`
	Status  string `json:"status,omitempty"`
}

// Simple in-memory notification store (for demo purposes)
var notifications = []NotificationMessage{
	{
		Type:    "info",
		Title:   "Giggum Started",
		Message: "Web server and notification system initialized",
		Status:  "active",
	},
}

// getNotifications returns all notifications
func (ws *WebServer) getNotifications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// sendNotification adds a new notification (for external systems)
func (ws *WebServer) sendNotification(c *gin.Context) {
	var notification NotificationMessage
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification format"})
		return
	}

	// Add timestamp (in real implementation)
	notifications = append(notifications, notification)

	// Keep only last 50 notifications
	if len(notifications) > 50 {
		notifications = notifications[len(notifications)-50:]
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "notification added",
		"total_count":  len(notifications),
		"notification": notification,
	})
}
