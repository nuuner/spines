package handlers

import (
	"math/rand"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

// UserWithCurrentBook combines a User with their currently reading book (if any)
type UserWithCurrentBook struct {
	models.User
	CurrentlyReading *models.UserBook
}

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

	// Collect user IDs for bulk query
	userIDs := make([]int64, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// Fetch one random currently reading book per user
	currentlyReading, err := models.GetRandomCurrentlyReadingByUserIDs(userIDs)
	if err != nil {
		currentlyReading = make(map[int64]*models.UserBook)
	}

	// Combine users with their currently reading books
	usersWithBooks := make([]UserWithCurrentBook, len(users))
	for i, u := range users {
		usersWithBooks[i] = UserWithCurrentBook{
			User:             u,
			CurrentlyReading: currentlyReading[u.ID],
		}
	}

	// Randomize user order on each page load
	rand.Shuffle(len(usersWithBooks), func(i, j int) {
		usersWithBooks[i], usersWithBooks[j] = usersWithBooks[j], usersWithBooks[i]
	})

	return c.Render("pages/dashboard", NavData(c, fiber.Map{
		"Users": usersWithBooks,
		// SEO metadata
		"PageTitle":       "Readers",
		"MetaDescription": "Discover book collections and reading lists on Spines",
		"OGTitle":         "Spines - Track Your Reading",
		"OGDescription":   "Discover book collections and reading lists on Spines",
		"OGType":          "website",
	}), "layouts/base")
}
