package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func AdminDashboard(c *fiber.Ctx) error {
	users, err := models.GetAllUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading users")
	}

	return c.Render("pages/admin/dashboard", NavData(c, fiber.Map{
		"Users": users,
	}), "layouts/base")
}

func AdminUsersList(c *fiber.Ctx) error {
	users, err := models.GetAllUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading users")
	}

	return c.Render("pages/admin/users", NavData(c, fiber.Map{
		"Users": users,
		"Error": c.Query("error"),
	}), "layouts/base")
}

func AdminCreateUser(c *fiber.Ctx) error {
	username := c.FormValue("username")
	displayName := c.FormValue("display_name")
	description := c.FormValue("description")

	if username == "" || displayName == "" {
		return c.Redirect("/admin/users?error=Username+and+display+name+are+required")
	}

	_, err := models.CreateUser(username, displayName, description)
	if err != nil {
		return c.Redirect("/admin/users?error=Failed+to+create+user")
	}

	return c.Redirect("/admin/users")
}

func AdminEditUser(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	user, err := models.GetUserByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading user")
	}

	return c.Render("pages/admin/edit_user", NavData(c, fiber.Map{
		"User":    user,
		"Error":   c.Query("error"),
		"Success": c.Query("success"),
	}), "layouts/base")
}

func AdminUpdateUser(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	username := c.FormValue("username")
	displayName := c.FormValue("display_name")
	description := c.FormValue("description")

	if username == "" || displayName == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Username+and+display+name+are+required")
	}

	err = models.UpdateUser(id, username, displayName, description)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Failed+to+update+user")
	}

	return c.Redirect("/admin/users")
}

func AdminDeleteUser(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	err = models.DeleteUser(id)
	if err != nil {
		return c.Redirect("/admin/users?error=Failed+to+delete+user")
	}

	return c.Redirect("/admin/users")
}

func AdminSetPassword(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	if password == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Password+is+required")
	}

	if password != confirmPassword {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Passwords+do+not+match")
	}

	if len(password) < 8 {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Password+must+be+at+least+8+characters")
	}

	err = models.SetUserPassword(id, password)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Failed+to+set+password")
	}

	return c.Redirect("/admin/users/" + c.Params("id") + "/edit?success=Password+set+successfully")
}

func AdminClearPassword(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	err = models.ClearUserPassword(id)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/edit?error=Failed+to+clear+password")
	}

	return c.Redirect("/admin/users/" + c.Params("id") + "/edit?success=Login+disabled+for+user")
}
