package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

// NavData merges navigation context into template data
func NavData(c *fiber.Ctx, data fiber.Map) fiber.Map {
	if data == nil {
		data = fiber.Map{}
	}
	if isAdmin, ok := c.Locals("IsAdmin").(bool); ok {
		data["IsAdmin"] = isAdmin
	}
	if user, ok := c.Locals("CurrentUser").(*models.User); ok {
		data["CurrentUser"] = user
	}
	// Add CSRF token for forms
	if csrfToken, ok := c.Locals("CSRFToken").(string); ok {
		data["CSRFToken"] = csrfToken
	}
	return data
}
