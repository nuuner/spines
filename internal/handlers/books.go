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
	thumbnailURL := c.FormValue("thumbnail_url")
	isbn13 := c.FormValue("isbn_13")
	isbn10 := c.FormValue("isbn_10")
	pageCount, _ := strconv.Atoi(c.FormValue("page_count"))
	shelf := c.FormValue("shelf")
	subStatus := c.FormValue("sub_status")

	if googleBooksID == "" || title == "" || shelf == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Missing+required+fields")
	}

	book, err := models.GetOrCreateBookWithISBN(googleBooksID, title, authors, thumbnailURL, isbn13, isbn10, pageCount, h.Config.GoogleBooksAPIKey)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+create+book")
	}

	var nullSubStatus sql.NullString
	if subStatus != "" {
		nullSubStatus = sql.NullString{String: subStatus, Valid: true}
	}

	err = models.AddBookToShelf(userID, book.ID, shelf, nullSubStatus)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+add+book+to+shelf")
	}

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

	if shelf == "" {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Shelf+is+required")
	}

	var nullSubStatus sql.NullString
	if subStatus != "" {
		nullSubStatus = sql.NullString{String: subStatus, Valid: true}
	}

	err = models.UpdateUserBook(userID, bookID, shelf, nullSubStatus)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+update+book")
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

	err = models.RemoveBookFromShelf(userID, bookID)
	if err != nil {
		return c.Redirect("/admin/users/" + c.Params("id") + "/books?error=Failed+to+remove+book")
	}

	return c.Redirect("/admin/users/" + c.Params("id") + "/books")
}
