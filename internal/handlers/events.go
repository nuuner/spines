package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/models"
)

// GetLatestEvents returns the most recent event for each user (for news feed)
func GetLatestEvents(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}

	events, err := models.GetLatestEventPerUser(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load events",
		})
	}

	// Transform events to a JSON-friendly format
	result := make([]fiber.Map, len(events))
	for i, ev := range events {
		eventData := fiber.Map{
			"id":          ev.ID,
			"event_type":  ev.EventType,
			"shelf":       ev.ShelfDisplay(),
			"old_value":   ev.OldValueDisplay(),
			"new_value":   ev.NewValueDisplay(),
			"description": ev.EventDescription(),
			"time_ago":    ev.TimeAgo(),
			"created_at":  ev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if ev.User != nil {
			eventData["user"] = fiber.Map{
				"id":              ev.User.ID,
				"username":        ev.User.Username,
				"display_name":    ev.User.DisplayName,
				"profile_picture": ev.User.GetProfilePictureURL(),
			}
		}

		if ev.Book != nil {
			eventData["book"] = fiber.Map{
				"id":            ev.Book.ID,
				"title":         ev.Book.Title,
				"authors":       ev.Book.Authors,
				"thumbnail_url": ev.Book.ThumbnailURL,
			}
		}

		result[i] = eventData
	}

	return c.JSON(fiber.Map{
		"events": result,
	})
}

// GetRecentEvents returns the most recent events across all users
func GetRecentEvents(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}

	events, err := models.GetRecentEvents(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load events",
		})
	}

	// Transform events to a JSON-friendly format
	result := make([]fiber.Map, len(events))
	for i, ev := range events {
		eventData := fiber.Map{
			"id":          ev.ID,
			"event_type":  ev.EventType,
			"shelf":       ev.ShelfDisplay(),
			"old_value":   ev.OldValueDisplay(),
			"new_value":   ev.NewValueDisplay(),
			"description": ev.EventDescription(),
			"time_ago":    ev.TimeAgo(),
			"created_at":  ev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if ev.User != nil {
			eventData["user"] = fiber.Map{
				"id":              ev.User.ID,
				"username":        ev.User.Username,
				"display_name":    ev.User.DisplayName,
				"profile_picture": ev.User.GetProfilePictureURL(),
			}
		}

		if ev.Book != nil {
			eventData["book"] = fiber.Map{
				"id":            ev.Book.ID,
				"title":         ev.Book.Title,
				"authors":       ev.Book.Authors,
				"thumbnail_url": ev.Book.ThumbnailURL,
			}
		}

		result[i] = eventData
	}

	return c.JSON(fiber.Map{
		"events": result,
	})
}

// GetUserEvents returns events for a specific user
func GetUserEvents(c *fiber.Ctx) error {
	username := c.Params("username")
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username is required",
		})
	}

	user, err := models.GetUserByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}

	events, err := models.GetUserEvents(user.ID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load events",
		})
	}

	// Transform events to a JSON-friendly format
	result := make([]fiber.Map, len(events))
	for i, ev := range events {
		eventData := fiber.Map{
			"id":          ev.ID,
			"event_type":  ev.EventType,
			"shelf":       ev.ShelfDisplay(),
			"old_value":   ev.OldValueDisplay(),
			"new_value":   ev.NewValueDisplay(),
			"description": ev.EventDescription(),
			"time_ago":    ev.TimeAgo(),
			"created_at":  ev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if ev.Book != nil {
			eventData["book"] = fiber.Map{
				"id":            ev.Book.ID,
				"title":         ev.Book.Title,
				"authors":       ev.Book.Authors,
				"thumbnail_url": ev.Book.ThumbnailURL,
			}
		}

		result[i] = eventData
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":              user.ID,
			"username":        user.Username,
			"display_name":    user.DisplayName,
			"profile_picture": user.GetProfilePictureURL(),
		},
		"events": result,
	})
}
