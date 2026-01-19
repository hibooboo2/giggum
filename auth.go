package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims structure for JWT tokens
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthConfig holds configuration for authentication
type AuthConfig struct {
	JWTSecret       string        // Secret key for JWT signing
	TokenExpiration time.Duration // Token expiration time
	RefreshTokenTTL time.Duration // Refresh token expiration time
	Issuer          string        // Token issuer
	EnableRefresh   bool          // Enable refresh tokens
}

// User represents a user in the system
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Password string   `json:"-"` // Password hash, not exposed in JSON
}

// AuthManager handles authentication and authorization
type AuthManager struct {
	config   AuthConfig
	users    map[string]*User      // In-memory user store (for demo)
	sessions map[string]*JWTClaims // Active sessions
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config AuthConfig) *AuthManager {
	// Generate random secret if not provided
	if config.JWTSecret == "" {
		secret := make([]byte, 32)
		rand.Read(secret)
		config.JWTSecret = base64.StdEncoding.EncodeToString(secret)
	}

	// Set default values
	if config.TokenExpiration == 0 {
		config.TokenExpiration = 24 * time.Hour // Default 24 hours
	}
	if config.RefreshTokenTTL == 0 {
		config.RefreshTokenTTL = 7 * 24 * time.Hour // Default 7 days
	}
	if config.Issuer == "" {
		config.Issuer = "giggum"
	}

	return &AuthManager{
		config:   config,
		users:    make(map[string]*User),
		sessions: make(map[string]*JWTClaims),
	}
}

// GenerateJWT generates a JWT token for a user
func (am *AuthManager) GenerateJWT(user *User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.config.TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    am.config.Issuer,
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.config.JWTSecret))
}

// ValidateJWT validates a JWT token and returns claims
func (am *AuthManager) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ExtractTokenFromRequest extracts JWT token from request header
func (am *AuthManager) ExtractTokenFromRequest(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	// Check for Bearer token
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer "), nil
	}

	// Check for token in query parameters (for WebSocket connections)
	token := c.Query("token")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("invalid authorization header format")
}

// CreateUser creates a new user (for demo purposes)
func (am *AuthManager) CreateUser(id, username, email, password string, roles []string) *User {
	user := &User{
		ID:       id,
		Username: username,
		Email:    email,
		Roles:    roles,
		Password: password, // In production, this should be a hash
	}

	am.users[id] = user
	return user
}

// GetUserByID retrieves a user by ID
func (am *AuthManager) GetUserByID(id string) (*User, bool) {
	user, exists := am.users[id]
	return user, exists
}

// HasRole checks if user has the required role
func (claims *JWTClaims) HasRole(role string) bool {
	for _, userRole := range claims.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if user has any of the required roles
func (claims *JWTClaims) HasAnyRole(roles ...string) bool {
	for _, requiredRole := range roles {
		if claims.HasRole(requiredRole) {
			return true
		}
	}
	return false
}

// AuthMiddleware creates authentication middleware
func (am *AuthManager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := am.ExtractTokenFromRequest(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": err.Error(),
				"code":    "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		claims, err := am.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
				"code":    "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Store claims in context for later use
		c.Set("user_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_roles", claims.Roles)

		c.Next()
	}
}

// RequireRole creates middleware that requires specific role
func (am *AuthManager) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsInterface, exists := c.Get("user_claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "user claims not found",
				"code":    "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		claims, ok := claimsInterface.(*JWTClaims)
		if !ok || !claims.HasRole(role) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("role '%s' required", role),
				"code":    "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles
func (am *AuthManager) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsInterface, exists := c.Get("user_claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "user claims not found",
				"code":    "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		claims, ok := claimsInterface.(*JWTClaims)
		if !ok || !claims.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": fmt.Sprintf("one of roles %v required", roles),
				"code":    "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRequest represents login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response payload
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// Login handles user authentication
func (am *AuthManager) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad request",
			"message": err.Error(),
			"code":    "INVALID_REQUEST",
		})
		return
	}

	// Find user by username (in production, you'd also check by email)
	var user *User
	for _, u := range am.users {
		if u.Username == req.Username {
			user = u
			break
		}
	}

	if user == nil || user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "invalid username or password",
			"code":    "INVALID_CREDENTIALS",
		})
		return
	}

	// Generate JWT token
	token, err := am.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal server error",
			"message": "failed to generate token",
			"code":    "TOKEN_GENERATION_FAILED",
		})
		return
	}

	expiresAt := time.Now().Add(am.config.TokenExpiration)

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: &User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Roles:    user.Roles,
		},
	})
}

// SetupAuthRoutes sets up authentication routes
func (am *AuthManager) SetupAuthRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", am.Login)
		// Add refresh token endpoint if enabled
		if am.config.EnableRefresh {
			auth.POST("/refresh", am.RefreshToken)
		}
		auth.GET("/me", am.AuthMiddleware(), am.GetCurrentUser)
	}
}

// GetCurrentUser returns the current authenticated user
func (am *AuthManager) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "user not authenticated",
			"code":    "AUTH_REQUIRED",
		})
		return
	}

	user, exists := am.GetUserByID(userID.(string))
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not found",
			"message": "user not found",
			"code":    "USER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": &User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Roles:    user.Roles,
		},
	})
}

// RefreshToken refreshes an existing JWT token
func (am *AuthManager) RefreshToken(c *gin.Context) {
	// Implementation for token refresh would go here
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "not implemented",
		"message": "token refresh not implemented",
		"code":    "NOT_IMPLEMENTED",
	})
}

// CreateDefaultAuthManager creates an auth manager with default configuration and demo users
func CreateDefaultAuthManager() *AuthManager {
	config := AuthConfig{
		JWTSecret:       "your-super-secret-jwt-key-change-this-in-production",
		TokenExpiration: 24 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "giggum",
		EnableRefresh:   false,
	}

	am := NewAuthManager(config)

	// Create demo users
	am.CreateUser("1", "admin", "admin@example.com", "admin123", []string{"admin"})
	am.CreateUser("2", "user", "user@example.com", "user123", []string{"user"})
	am.CreateUser("3", "developer", "dev@example.com", "dev123", []string{"user", "developer"})

	return am
}
