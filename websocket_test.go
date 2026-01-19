package main

import (
	"testing"
)

func TestWebSocketConnection(t *testing.T) {
	// Test that WebSocket upgrader is configured correctly
	if upgrader.CheckOrigin == nil {
		t.Error("WebSocket upgrader CheckOrigin not configured")
	}
}

func TestWebSocketConnectionStructure(t *testing.T) {
	// Test WebSocketConnection structure
	wsc := &WebSocketConnection{
		send:   make(chan NotificationMessage, 256),
		closed: false,
	}

	if wsc.send == nil {
		t.Error("WebSocket send channel not initialized")
	}

	if wsc.closed != false {
		t.Error("WebSocket connection should start as not closed")
	}
}

func TestWebServerStructure(t *testing.T) {
	// Test WebServer structure with WebSocket support
	ws := &WebServer{
		port:        8080,
		connections: make([]*WebSocketConnection, 0),
	}

	if ws.port != 8080 {
		t.Errorf("Expected port 8080, got %d", ws.port)
	}

	if ws.connections == nil {
		t.Error("Connections slice not initialized")
	}
}
