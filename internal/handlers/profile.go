package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func ProfilePage(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	return c.Render("pages/user/profile", NavData(c, fiber.Map{
		"User":    user,
		"Error":   c.Query("error"),
		"Success": c.Query("success"),
		// SEO metadata
		"PageTitle":  "My Profile",
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func UpdateProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	displayName := c.FormValue("display_name")
	description := c.FormValue("description")

	if displayName == "" {
		return c.Redirect("/profile?error=Display+name+is+required")
	}

	err := models.UpdateUserProfile(user.ID, displayName, description)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+update+profile")
	}

	return c.Redirect("/profile?success=Profile+updated+successfully")
}

func ChangePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	currentPassword := c.FormValue("current_password")
	newPassword := c.FormValue("new_password")
	confirmPassword := c.FormValue("confirm_password")

	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		return c.Redirect("/profile?error=All+password+fields+are+required")
	}

	if newPassword != confirmPassword {
		return c.Redirect("/profile?error=New+passwords+do+not+match")
	}

	if len(newPassword) < 8 {
		return c.Redirect("/profile?error=Password+must+be+at+least+8+characters")
	}

	if !user.CheckPassword(currentPassword) {
		return c.Redirect("/profile?error=Current+password+is+incorrect")
	}

	err := models.SetUserPassword(user.ID, newPassword)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+change+password")
	}

	return c.Redirect("/profile?success=Password+changed+successfully")
}
