package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/config"
	"github.com/nuuner/spines/internal/models"
	"github.com/nuuner/spines/internal/services"
)

type BooksHandler struct {
	Config *config.Config
}

func NewBooksHandler(cfg *config.Config) *BooksHandler {
	return &BooksHandler{Config: cfg}
}

func (h *BooksHandler) ManageUserBooks(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	user, err := models.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading user")
	}

	shelves, err := models.GetUserBooks(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading books")
	}

	return c.Render("pages/admin/books", NavData(c, fiber.Map{
		"User":    user,
		"Shelves": shelves,
		"Error":   c.Query("error"),
		// SEO metadata
		"PageTitle":  "Manage Books - " + user.DisplayName,
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func (h *BooksHandler) SearchBooks(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	user, err := models.GetUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading user")
	}

	isHtmx := c.Get("HX-Request") == "true"
	query := c.Query("q")
	var results []services.BookSearchResult

	if query != "" {
		results, err = services.SearchBooks(query, h.Config.GoogleBooksAPIKey)
		if err != nil {
			if isHtmx {
				return c.Render("partials/admin_search_results", fiber.Map{
					"User":  user,
					"Query": query,
					"Error": "Failed to search books: " + err.Error(),
				})
			}
			return c.Render("pages/admin/search", NavData(c, fiber.Map{
				"User":    user,
				"Query":   query,
				"Error":   "Failed to search books: " + err.Error(),
				"Results": nil,
				// SEO metadata
				"PageTitle":  "Search Books - " + user.DisplayName,
				"MetaRobots": "noindex, nofollow",
			}), "layouts/base")
		}
	}

	if isHtmx {
		return c.Render("partials/admin_search_results", fiber.Map{
			"User":    user,
			"Query":   query,
			"Results": results,
		})
	}

	return c.Render("pages/admin/search", NavData(c, fiber.Map{
		"User":    user,
		"Query":   query,
		"Results": results,
		// SEO metadata
		"PageTitle":  "Search Books - " + user.DisplayName,
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func (h *BooksHandler) AddBook(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	googleBooksID := c.FormValue("google_books_id")
	title := c.FormValue("title")
	authors := c.FormValue("authors")
	description := c.FormValue("description")
	thumbnailURL := c.FormValue("thumbnail_url")
	isbn13 := c.FormValue("isbn_13")
	isbn10 := c.FormValue("isbn_10")
	pageCount, _ := strconv.Atoi(c.FormValue("page_count"))
	shelf := c.FormValue("shelf")
	subStatus := c.FormValue("sub_status")
	ratingStr := c.FormValue("rating")

	if googleBooksID == "" || title == "" || shelf == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Missing+required+fields")
	}

	book, err := models.GetOrCreateBookWithISBN(googleBooksID, title, authors, description, thumbnailURL, isbn13, isbn10, pageCount, h.Config.GoogleBooksAPIKey)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+create+book")
	}

	var nullSubStatus sql.NullString
	if subStatus != "" {
		nullSubStatus = sql.NullString{String: subStatus, Valid: true}
	}

	var nullRating sql.NullInt64
	if ratingStr != "" {
		rating, err := strconv.ParseInt(ratingStr, 10, 64)
		if err == nil && rating >= 1 && rating <= 5 {
			nullRating = sql.NullInt64{Int64: rating, Valid: true}
		}
	}

	err = models.AddBookToShelf(userID, book.ID, shelf, nullSubStatus, nullRating)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+add+book+to+shelf")
	}

	// Create event for book added to shelf
	_ = models.CreateBookAddedEvent(userID, book.ID, shelf)

	return c.Redirect("/admin/users/" + c.Params("id") + "/books")
}

func (h *BooksHandler) UpdateBook(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	bookID, err := strconv.ParseInt(c.Params("book_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	shelf := c.FormValue("shelf")
	subStatus := c.FormValue("sub_status")
	ratingStr := c.FormValue("rating")

	if shelf == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Shelf+is+required")
	}

	// Get current state before updating
	currentBook, _ := models.GetUserBook(userID, bookID)
	oldShelf := ""
	oldSubStatus := ""
	if currentBook != nil {
		oldShelf = currentBook.Shelf
		if currentBook.SubStatus.Valid {
			oldSubStatus = currentBook.SubStatus.String
		}
	}

	var nullSubStatus sql.NullString
	if subStatus != "" {
		nullSubStatus = sql.NullString{String: subStatus, Valid: true}
	}

	var nullRating sql.NullInt64
	if ratingStr != "" {
		rating, err := strconv.ParseInt(ratingStr, 10, 64)
		if err == nil && rating >= 1 && rating <= 5 {
			nullRating = sql.NullInt64{Int64: rating, Valid: true}
		}
	}

	err = models.UpdateUserBook(userID, bookID, shelf, nullSubStatus, nullRating)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+update+book")
	}

	// Create appropriate event based on what changed
	if currentBook != nil {
		if oldShelf != shelf {
			// Shelf changed - create book moved event
			_ = models.CreateBookMovedEvent(userID, bookID, oldShelf, shelf)
		} else if oldSubStatus != subStatus && shelf == "currently_reading" {
			// Reading progress changed
			_ = models.CreateReadingProgressEvent(userID, bookID, oldSubStatus, subStatus)
		}
	}

	return c.Redirect("/admin/users/" + c.Params("id") + "/books")
}

func (h *BooksHandler) RemoveBook(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid user ID")
	}

	bookID, err := strconv.ParseInt(c.Params("book_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	// Get current state before removing (for the event)
	currentBook, _ := models.GetUserBook(userID, bookID)
	oldShelf := ""
	if currentBook != nil {
		oldShelf = currentBook.Shelf
	}

	err = models.RemoveBookFromShelf(userID, bookID)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+remove+book")
	}

	// Create event for book removed
	if oldShelf != "" {
		_ = models.CreateBookRemovedEvent(userID, bookID, oldShelf)
	}

	return c.Redirect("/admin/users/" + c.Params("id") + "/books")
}

// BackfillDescriptions fetches descriptions from Google Books API for all books without descriptions
func (h *BooksHandler) BackfillDescriptions(c *fiber.Ctx) error {
	books, err := models.GetBooksWithoutDescription()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get books: " + err.Error(),
		})
	}

	updated := 0
	failed := 0
	skipped := 0

	for _, book := range books {
		if book.GoogleBooksID == "" {
			skipped++
			continue
		}

		result, err := services.GetBookByGoogleBooksID(book.GoogleBooksID, h.Config.GoogleBooksAPIKey)
		if err != nil {
			failed++
			continue
		}

		if result.Description == "" {
			skipped++
			continue
		}

		err = models.UpdateBookDescription(book.ID, result.Description)
		if err != nil {
			failed++
			continue
		}

		updated++
	}

	return c.JSON(fiber.Map{
		"total":   len(books),
		"updated": updated,
		"failed":  failed,
		"skipped": skipped,
	})
}
