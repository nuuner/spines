package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

func Dashboard(c *fiber.Ctx) error {
	users, err := models.GetAllUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading users")
	}

	if len(users) == 0 {
		return c.Redirect("/admin")
	}

	if len(users) == 1 {
		return c.Redirect("/u/" + users[0].Username)
	}

	return c.Render("pages/dashboard", NavData(c, fiber.Map{
		"Users": users,
	}), "layouts/base")
}
