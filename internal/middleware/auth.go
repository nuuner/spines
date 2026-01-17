package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func AdminAuth(c *fiber.Ctx) error {
	token := c.Cookies("admin_session_token")
	if token == "" {
		return c.Redirect("/admin/login")
	}

	if !models.IsValidAdminSession(token) {
		c.ClearCookie("admin_session_token")
		return c.Redirect("/admin/login")
	}

	return c.Next()
}

func UserAuth(c *fiber.Ctx) error {
	token := c.Cookies("user_session_token")
	if token == "" {
		return c.Redirect("/login")
	}

	userID, valid := models.IsValidUserSession(token)
	if !valid {
		c.ClearCookie("user_session_token")
		return c.Redirect("/login")
	}

	// Load user and store in context
	user, err := models.GetUserByID(userID)
	if err != nil {
		c.ClearCookie("user_session_token")
		return c.Redirect("/login")
	}

	c.Locals("user", user)
	return c.Next()
}
