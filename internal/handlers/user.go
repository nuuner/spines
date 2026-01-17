package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func UserPage(c *fiber.Ctx) error {
	username := c.Params("username")

	user, err := models.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading user")
	}

	shelves, err := models.GetUserBooks(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading books")
	}

	return c.Render("pages/user", NavData(c, fiber.Map{
		"User":    user,
		"Shelves": shelves,
	}), "layouts/base")
}
