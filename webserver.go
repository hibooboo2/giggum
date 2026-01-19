package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketConnection represents a connected WebSocket client
type WebSocketConnection struct {
	conn   *websocket.Conn
	send   chan NotificationMessage
	mutex  sync.Mutex
	closed bool
}

// WebServer represents the HTTP server for giggum PWA integration
type WebServer struct {
	port           int
	connections    []*WebSocketConnection
	connectionsMux sync.RWMutex
}

// NewWebServer creates a new web server instance
func NewWebServer(port int) *WebServer {
	return &WebServer{
		port:        port,
		connections: make([]*WebSocketConnection, 0),
	}
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

	// WebSocket endpoint for real-time notifications
	router.GET("/ws", ws.handleWebSocket)

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

	log.Println("notifications: ", notification)
	log.Println(len(notifications))

	c.JSON(http.StatusOK, gin.H{
		"status":       "notification added",
		"total_count":  len(notifications),
		"notification": notification,
	})

	// Broadcast the new notification to all connected WebSocket clients
	ws.broadcastNotification(notification)
}

// handleWebSocket handles WebSocket connections for real-time notifications
func (ws *WebServer) handleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Create new connection
	wsConn := &WebSocketConnection{
		conn:   conn,
		send:   make(chan NotificationMessage, 256),
		closed: false,
	}

	// Add connection to the list
	ws.connectionsMux.Lock()
	ws.connections = append(ws.connections, wsConn)
	ws.connectionsMux.Unlock()

	log.Printf("New WebSocket connection established. Total connections: %d", len(ws.connections))

	// Start goroutines for reading and writing
	go wsConn.writePump()
	go wsConn.readPump()

}

// writePump handles sending messages to WebSocket
func (wsc *WebSocketConnection) writePump() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		wsc.conn.Close()
	}()

	for {
		select {
		case message, ok := <-wsc.send:
			wsc.mutex.Lock()
			if wsc.closed {
				wsc.mutex.Unlock()
				return
			}
			if !ok {
				wsc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				wsc.mutex.Unlock()
				return
			}

			if err := wsc.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				wsc.mutex.Unlock()
				return
			}
			wsc.mutex.Unlock()

		case <-ticker.C:
			wsc.mutex.Lock()
			if wsc.closed {
				wsc.mutex.Unlock()
				return
			}
			if err := wsc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error: %v", err)
				wsc.mutex.Unlock()
				return
			}
			wsc.mutex.Unlock()
		}
	}
}

// readPump handles reading messages from WebSocket
func (wsc *WebSocketConnection) readPump() {
	defer func() {
		wsc.mutex.Lock()
		wsc.closed = true
		wsc.mutex.Unlock()
		wsc.conn.Close()
	}()

	wsc.conn.SetReadLimit(512)
	wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsc.conn.SetPongHandler(func(string) error {
		wsc.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := wsc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// broadcastNotification sends a notification to all connected WebSocket clients
func (ws *WebServer) broadcastNotification(notification NotificationMessage) {
	ws.connectionsMux.RLock()
	defer ws.connectionsMux.RUnlock()

	for _, conn := range ws.connections {
		select {
		case conn.send <- notification:
		default:
			// Can't send, connection is probably closed
			close(conn.send)
		}
	}
}

// cleanupConnections removes closed connections
func (ws *WebServer) cleanupConnections() {
	ws.connectionsMux.Lock()
	defer ws.connectionsMux.Unlock()

	var activeConnections []*WebSocketConnection
	for _, conn := range ws.connections {
		conn.mutex.Lock()
		if !conn.closed {
			activeConnections = append(activeConnections, conn)
		}
		conn.mutex.Unlock()
	}

	ws.connections = activeConnections
}
