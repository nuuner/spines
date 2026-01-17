package handlers

import (
	"database/sql"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func UserLoginPage(c *fiber.Ctx) error {
	token := c.Cookies("user_session_token")
	if token != "" {
		if userID, valid := models.IsValidUserSession(token); valid {
			user, err := models.GetUserByID(userID)
			if err == nil {
				return c.Redirect("/u/" + user.Username)
			}
		}
	}

	return c.Render("pages/user/login", NavData(c, fiber.Map{
		"Error": c.Query("error"),
	}), "layouts/base")
}

func UserLogin(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Redirect("/login?error=Username+and+password+are+required")
	}

	user, err := models.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("SECURITY: Failed login attempt for unknown user '%s' from IP %s", username, c.IP())
			return c.Redirect("/login?error=Invalid+username+or+password")
		}
		return c.Redirect("/login?error=An+error+occurred")
	}

	if !user.HasPassword() {
		log.Printf("SECURITY: Login attempt for user '%s' with no password set from IP %s", username, c.IP())
		return c.Redirect("/login?error=Login+not+enabled+for+this+account")
	}

	if !user.CheckPassword(password) {
		log.Printf("SECURITY: Failed login attempt for user '%s' from IP %s", username, c.IP())
		return c.Redirect("/login?error=Invalid+username+or+password")
	}

	log.Printf("SECURITY: Successful login for user '%s' from IP %s", username, c.IP())

	session, err := models.CreateUserSession(user.ID)
	if err != nil {
		return c.Redirect("/login?error=Failed+to+create+session")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "user_session_token",
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect("/my-books")
}

func UserLogout(c *fiber.Ctx) error {
	token := c.Cookies("user_session_token")
	if token != "" {
		log.Printf("SECURITY: User logout from IP %s", c.IP())
		models.DeleteSession(token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "user_session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect("/login")
}
