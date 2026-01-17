package handlers

import (
	"crypto/subtle"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/config"
	"github.com/nuuner/spines/internal/models"
)

type AuthHandler struct {
	Config *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{Config: cfg}
}

func (h *AuthHandler) LoginPage(c *fiber.Ctx) error {
	token := c.Cookies("admin_session_token")
	if token != "" && models.IsValidAdminSession(token) {
		return c.Redirect("/admin")
	}

	return c.Render("pages/admin/login", NavData(c, fiber.Map{
		"Error": c.Query("error"),
	}), "layouts/base")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	password := c.FormValue("password")

	if h.Config.AdminPassword == "" {
		log.Printf("SECURITY: Admin login attempt with unconfigured password from IP %s", c.IP())
		return c.Redirect("/admin/login?error=Admin+password+not+configured")
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(password), []byte(h.Config.AdminPassword)) != 1 {
		log.Printf("SECURITY: Failed admin login attempt from IP %s", c.IP())
		return c.Redirect("/admin/login?error=Invalid+password")
	}

	log.Printf("SECURITY: Successful admin login from IP %s", c.IP())

	session, err := models.CreateAdminSession()
	if err != nil {
		return c.Redirect("/admin/login?error=Failed+to+create+session")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "admin_session_token",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect("/admin")
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	token := c.Cookies("admin_session_token")
	if token != "" {
		log.Printf("SECURITY: Admin logout from IP %s", c.IP())
		models.DeleteSession(token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "admin_session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect("/admin/login")
}
