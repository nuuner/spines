package handlers

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/config"
	"github.com/nuuner/spines/internal/models"
	"github.com/nuuner/spines/internal/services"
)

type UserBooksHandler struct {
	Config *config.Config
}

// ValidShelves defines the allowed shelf values
var ValidShelves = map[string]bool{
	"want_to_read":      true,
	"currently_reading": true,
	"read":              true,
}

// isValidShelf checks if the provided shelf name is valid
func isValidShelf(shelf string) bool {
	return ValidShelves[shelf]
}

func NewUserBooksHandler(cfg *config.Config) *UserBooksHandler {
	return &UserBooksHandler{Config: cfg}
}

func (h *UserBooksHandler) MyBooks(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	shelves, err := models.GetUserBooks(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading books")
	}

	return c.Render("pages/user/my_books", NavData(c, fiber.Map{
		"User":    user,
		"Shelves": shelves,
		"Error":   c.Query("error"),
		// SEO metadata
		"PageTitle":  "My Books",
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func (h *UserBooksHandler) SearchBooks(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	isHtmx := c.Get("HX-Request") == "true"

	query := c.Query("q")
	var results []services.BookSearchResult
	var err error

	if query != "" {
		results, err = services.SearchBooks(query, h.Config.GoogleBooksAPIKey)
		if err != nil {
			if isHtmx {
				return c.Render("partials/user_search_results", fiber.Map{
					"Query": query,
					"Error": "Failed to search books: " + err.Error(),
				})
			}
			return c.Render("pages/user/search", NavData(c, fiber.Map{
				"User":    user,
				"Query":   query,
				"Error":   "Failed to search books: " + err.Error(),
				"Results": nil,
				// SEO metadata
				"PageTitle":  "Search Books",
				"MetaRobots": "noindex, nofollow",
			}), "layouts/base")
		}
	}

	if isHtmx {
		return c.Render("partials/user_search_results", fiber.Map{
			"Query":   query,
			"Results": results,
		})
	}

	return c.Render("pages/user/search", NavData(c, fiber.Map{
		"User":    user,
		"Query":   query,
		"Results": results,
		// SEO metadata
		"PageTitle":  "Search Books",
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func (h *UserBooksHandler) AddBookPage(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	googleBooksID := c.Query("google_books_id")
	title := c.Query("title")
	authors := c.Query("authors")
	description := c.Query("description")
	thumbnailURL := c.Query("thumbnail_url")
	isbn13 := c.Query("isbn_13")
	isbn10 := c.Query("isbn_10")
	pageCount := c.QueryInt("page_count", 0)
	query := c.Query("q")

	if googleBooksID == "" || title == "" {
		return c.Redirect("/my-books/search")
	}

	return c.Render("pages/user/add_book", NavData(c, fiber.Map{
		"User":          user,
		"GoogleBooksID": googleBooksID,
		"Title":         title,
		"Authors":       authors,
		"Description":   description,
		"ThumbnailURL":  thumbnailURL,
		"ISBN13":        isbn13,
		"ISBN10":        isbn10,
		"PageCount":     pageCount,
		"Query":         query,
		// SEO metadata
		"PageTitle":  "Add Book",
		"MetaRobots": "noindex, nofollow",
	}), "layouts/base")
}

func (h *UserBooksHandler) AddBook(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

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
		return c.Redirect("/my-books?error=Missing+required+fields")
	}

	if !isValidShelf(shelf) {
		return c.Redirect("/my-books?error=Invalid+shelf")
	}

	book, err := models.GetOrCreateBookWithISBN(googleBooksID, title, authors, description, thumbnailURL, isbn13, isbn10, pageCount, h.Config.GoogleBooksAPIKey)
	if err != nil {
		return c.Redirect("/my-books?error=Failed+to+create+book")
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

	err = models.AddBookToShelf(user.ID, book.ID, shelf, nullSubStatus, nullRating)
	if err != nil {
		return c.Redirect("/my-books?error=Failed+to+add+book+to+shelf")
	}

	// Create event for book added to shelf
	_ = models.CreateBookAddedEvent(user.ID, book.ID, shelf)

	return c.Redirect("/my-books")
}

func (h *UserBooksHandler) UpdateBook(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	bookID, err := strconv.ParseInt(c.Params("book_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	if bookID <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	shelf := c.FormValue("shelf")
	subStatus := c.FormValue("sub_status")
	ratingStr := c.FormValue("rating")

	if shelf == "" {
		return c.Redirect("/my-books?error=Shelf+is+required")
	}

	if !isValidShelf(shelf) {
		return c.Redirect("/my-books?error=Invalid+shelf")
	}

	// Get current state before updating
	currentBook, _ := models.GetUserBook(user.ID, bookID)
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

	err = models.UpdateUserBook(user.ID, bookID, shelf, nullSubStatus, nullRating)
	if err != nil {
		return c.Redirect("/my-books?error=Failed+to+update+book")
	}

	// Create appropriate event based on what changed
	if currentBook != nil {
		if oldShelf != shelf {
			// Shelf changed - create book moved event
			_ = models.CreateBookMovedEvent(user.ID, bookID, oldShelf, shelf)
		} else if oldSubStatus != subStatus && shelf == "currently_reading" {
			// Reading progress changed
			_ = models.CreateReadingProgressEvent(user.ID, bookID, oldSubStatus, subStatus)
		}
	}

	return c.Redirect("/my-books")
}

func (h *UserBooksHandler) RemoveBook(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	bookID, err := strconv.ParseInt(c.Params("book_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	// Get current state before removing (for the event)
	currentBook, _ := models.GetUserBook(user.ID, bookID)
	oldShelf := ""
	if currentBook != nil {
		oldShelf = currentBook.Shelf
	}

	err = models.RemoveBookFromShelf(user.ID, bookID)
	if err != nil {
		return c.Redirect("/my-books?error=Failed+to+remove+book")
	}

	// Create event for book removed
	if oldShelf != "" {
		_ = models.CreateBookRemovedEvent(user.ID, bookID, oldShelf)
	}

	return c.Redirect("/my-books")
}

func (h *UserBooksHandler) UpdateBookDates(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	bookID, err := strconv.ParseInt(c.Params("book_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid book ID")
	}

	// Convert datetime-local format (2006-01-02T15:04) to SQLite format (2006-01-02 15:04:05)
	parseDateTime := func(value string) sql.NullString {
		if value == "" {
			return sql.NullString{}
		}
		t, err := time.ParseInLocation("2006-01-02T15:04", value, time.Local)
		if err != nil {
			return sql.NullString{}
		}
		return sql.NullString{String: t.Format("2006-01-02 15:04:05"), Valid: true}
	}

	addedAt := parseDateTime(c.FormValue("added_at"))
	startedReadingAt := parseDateTime(c.FormValue("started_reading_at"))
	finishedReadingAt := parseDateTime(c.FormValue("finished_reading_at"))

	err = models.UpdateUserBookDates(user.ID, bookID, addedAt, startedReadingAt, finishedReadingAt)
	if err != nil {
		return c.Redirect("/my-books?error=Failed+to+update+dates")
	}

	return c.Redirect("/my-books")
}

func (h *UserBooksHandler) GetShelfBooks(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	shelf := c.Params("shelf")

	if !isValidShelf(shelf) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid shelf")
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

	return c.Render("partials/shelf_books", fiber.Map{
		"Books":      books,
		"Shelf":      shelf,
		"NextOffset": offset + len(books),
		"Remaining":  remaining,
	})
}
