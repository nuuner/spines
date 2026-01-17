package middleware

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Rate limiter for authentication endpoints
type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string][]time.Time
	window   time.Duration
	limit    int
}

func NewRateLimiter(window time.Duration, limit int) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		limit:    limit,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.attempts {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) IsAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.attempts[ip]

	// Filter out old attempts
	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.attempts[ip] = valid
		return false
	}

	valid = append(valid, now)
	rl.attempts[ip] = valid
	return true
}

// Global rate limiter for auth endpoints: 10 attempts per 15 minutes
var AuthRateLimiter = NewRateLimiter(15*time.Minute, 10)

// RateLimitAuth middleware for authentication endpoints
func RateLimitAuth(c *fiber.Ctx) error {
	ip := c.IP()

	if !AuthRateLimiter.IsAllowed(ip) {
		log.Printf("SECURITY: Rate limit exceeded for IP %s on %s", ip, c.Path())
		return c.Status(fiber.StatusTooManyRequests).SendString("Too many login attempts. Please try again later.")
	}

	return c.Next()
}

// CSRF token management
const (
	CSRFTokenLength   = 32
	CSRFCookieName    = "csrf_token"
	CSRFFormFieldName = "csrf_token"
	CSRFHeaderName    = "X-CSRF-Token"
)

// GenerateCSRFToken creates a new CSRF token
func GenerateCSRFToken() string {
	return uuid.New().String()
}

// CSRFProtection middleware adds CSRF protection to POST/PUT/DELETE requests
func CSRFProtection(c *fiber.Ctx) error {
	// GET requests: ensure token exists
	if c.Method() == "GET" {
		token := c.Cookies(CSRFCookieName)
		if token == "" {
			token = GenerateCSRFToken()
			c.Cookie(&fiber.Cookie{
				Name:     CSRFCookieName,
				Value:    token,
				HTTPOnly: false, // Must be readable by forms
				SameSite: "Lax",
				Secure:   false, // Set to true in production with HTTPS
				Path:     "/",
			})
		}
		c.Locals("CSRFToken", token)
		return c.Next()
	}

	// POST/PUT/DELETE: validate token
	if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "DELETE" {
		cookieToken := c.Cookies(CSRFCookieName)
		formToken := c.FormValue(CSRFFormFieldName)
		headerToken := c.Get(CSRFHeaderName)

		// Accept token from either form or header
		submittedToken := formToken
		if submittedToken == "" {
			submittedToken = headerToken
		}

		if cookieToken == "" || submittedToken == "" || cookieToken != submittedToken {
			log.Printf("SECURITY: CSRF validation failed for IP %s on %s", c.IP(), c.Path())
			return c.Status(fiber.StatusForbidden).SendString("Invalid or missing CSRF token")
		}

		c.Locals("CSRFToken", cookieToken)
	}

	return c.Next()
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(c *fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("X-XSS-Protection", "1; mode=block")
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	return c.Next()
}
