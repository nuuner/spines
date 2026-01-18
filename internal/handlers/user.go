package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

// Number of books to show initially on public user page per shelf
const publicShelfInitialLimit = 8

// validPublicShelves defines the allowed shelf values for public pages
var validPublicShelves = map[string]bool{
	"want_to_read": true,
	"read":         true,
}

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

	// Build meta description
	metaDesc := user.DisplayName + "'s book collection on Spines"
	if user.Description != "" {
		metaDesc = user.Description
	}

	// Count total books for description enhancement
	totalBooks := len(shelves.WantToRead) + len(shelves.Read) + len(shelves.CurrentlyReading)
	if totalBooks > 0 && user.Description == "" {
		metaDesc = user.DisplayName + " has " + formatBookCount(totalBooks) + " on their reading list"
	}

	return c.Render("pages/user", NavData(c, fiber.Map{
		"User":                    user,
		"Shelves":                 shelves,
		"WantToReadTotal":         len(shelves.WantToRead),
		"ReadTotal":               len(shelves.Read),
		"PublicShelfInitialLimit": publicShelfInitialLimit,
		// SEO metadata
		"PageTitle":       user.DisplayName,
		"MetaDescription": metaDesc,
		"OGTitle":         user.DisplayName + " - Spines",
		"OGDescription":   metaDesc,
		"OGImage":         user.GetProfilePictureURL(),
		"OGType":          "profile",
	}), "layouts/base")
}

func formatBookCount(count int) string {
	if count == 1 {
		return "1 book"
	}
	return strconv.Itoa(count) + " books"
}

func GetPublicShelfBooks(c *fiber.Ctx) error {
	username := c.Params("username")
	shelf := c.Params("shelf")

	if !validPublicShelves[shelf] {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid shelf")
	}

	user, err := models.GetUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading user")
	}

	offset := c.QueryInt("offset", 0)
	limit := 20 // load 20 more each time

	books, total, err := models.GetShelfBooksPaginated(user.ID, shelf, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading books")
	}

	remaining := total - offset - len(books)
	if remaining < 0 {
		remaining = 0
	}

	return c.Render("partials/public_shelf_books", fiber.Map{
		"Books":      books,
		"Shelf":      shelf,
		"Username":   username,
		"NextOffset": offset + len(books),
		"Remaining":  remaining,
	})
}
