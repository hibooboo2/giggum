package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	RequestsPerSecond int           // Number of requests allowed per second
	BurstSize         int           // Maximum burst size
	CleanupInterval   time.Duration // Interval to clean up old entries
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate int
	lastRefill time.Time
	mutex      sync.Mutex
}

// RateLimiter manages multiple token buckets per client
type RateLimiter struct {
	buckets     map[string]*TokenBucket
	config      RateLimiterConfig
	mutex       sync.RWMutex
	lastCleanup time.Time
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens, refillRate int) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// ConsumeToken attempts to consume a token from the bucket
func (tb *TokenBucket) ConsumeToken() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate

	if tokensToAdd > 0 {
		tb.tokens = min(tb.maxTokens, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Check if we have a token available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 10 // Default to 10 requests per second
	}
	if config.BurstSize <= 0 {
		config.BurstSize = config.RequestsPerSecond * 2 // Default burst to 2x rate
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute // Default cleanup interval
	}

	return &RateLimiter{
		buckets:     make(map[string]*TokenBucket),
		config:      config,
		lastCleanup: time.Now(),
	}
}

// getClientKey extracts a client key from the request
func (rl *RateLimiter) getClientKey(c *gin.Context) string {
	// Use IP address as client identifier for simplicity
	// In production, you might want to use authenticated user ID
	clientIP := c.ClientIP()
	return clientIP
}

// IsAllowed checks if a request is allowed based on rate limiting
func (rl *RateLimiter) IsAllowed(c *gin.Context) bool {
	clientKey := rl.getClientKey(c)

	rl.mutex.RLock()
	bucket, exists := rl.buckets[clientKey]
	rl.mutex.RUnlock()

	if !exists {
		// Create new bucket for this client
		bucket = NewTokenBucket(rl.config.BurstSize, rl.config.RequestsPerSecond)

		rl.mutex.Lock()
		rl.buckets[clientKey] = bucket
		rl.mutex.Unlock()
	}

	// Clean up old buckets periodically
	rl.cleanupOldBuckets()

	return bucket.ConsumeToken()
}

// cleanupOldBuckets removes buckets that haven't been used recently
func (rl *RateLimiter) cleanupOldBuckets() {
	if time.Since(rl.lastCleanup) < rl.config.CleanupInterval {
		return
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	for key, bucket := range rl.buckets {
		bucket.mutex.Lock()
		// Remove bucket if it's been inactive for more than cleanup interval
		if time.Since(bucket.lastRefill) > rl.config.CleanupInterval {
			delete(rl.buckets, key)
		}
		bucket.mutex.Unlock()
	}

	rl.lastCleanup = time.Now()
}

// RateLimitMiddleware returns a Gin middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.IsAllowed(c) {
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			c.Header("Retry-After", "1")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
		// Note: We can't easily determine remaining tokens, so we'll leave it out
		// In a more sophisticated implementation, you could track this

		c.Next()
	}
}

// CreateRateLimitMiddleware creates a rate limit middleware with default configuration
func CreateRateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	config := RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         requestsPerSecond * 2,
		CleanupInterval:   5 * time.Minute,
	}

	limiter := NewRateLimiter(config)
	return limiter.RateLimitMiddleware()
}

// CreateGlobalRateLimitMiddleware creates a rate limit middleware that applies globally
func CreateGlobalRateLimitMiddleware() gin.HandlerFunc {
	// Default rate limit: 100 requests per second
	return CreateRateLimitMiddleware(100)
}

// CreateAPIRateLimitMiddleware creates a stricter rate limit for API endpoints
func CreateAPIRateLimitMiddleware() gin.HandlerFunc {
	// API rate limit: 50 requests per second
	return CreateRateLimitMiddleware(50)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
