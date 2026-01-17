package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

// NavContext sets navigation context for all routes
func NavContext(c *fiber.Ctx) error {
	// Check admin session
	if token := c.Cookies("admin_session_token"); token != "" {
		if models.IsValidAdminSession(token) {
			c.Locals("IsAdmin", true)
		}
	}

	// Check user session
	if token := c.Cookies("user_session_token"); token != "" {
		if userID, valid := models.IsValidUserSession(token); valid {
			if user, err := models.GetUserByID(userID); err == nil {
				c.Locals("CurrentUser", user)
			}
		}
	}

	return c.Next()
}
