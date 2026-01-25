package models

import (
	"database/sql"
	"log"
	"time"

	"github.com/nuuner/spines/internal/database"
	"github.com/nuuner/spines/internal/services"
)

type Book struct {
	ID            int64
	GoogleBooksID string
	Title         string
	Authors       string
	Description   sql.NullString
	ThumbnailURL  string
	ISBN13        sql.NullString
	ISBN10        sql.NullString
	PageCount     sql.NullInt64
	CreatedAt     time.Time
}

func GetBookByGoogleID(googleBooksID string) (*Book, error) {
	var b Book
	err := database.DB.QueryRow(
		"SELECT id, google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count, created_at FROM books WHERE google_books_id = ?",
		googleBooksID,
	).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.Description, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func GetBookByID(id int64) (*Book, error) {
	var b Book
	err := database.DB.QueryRow(
		"SELECT id, google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count, created_at FROM books WHERE id = ?",
		id,
	).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.Description, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func CreateBook(googleBooksID, title, authors, description, thumbnailURL, isbn13, isbn10 string, pageCount int) (int64, error) {
	var nullISBN13, nullISBN10, nullDescription sql.NullString
	var nullPageCount sql.NullInt64

	if isbn13 != "" {
		nullISBN13 = sql.NullString{String: isbn13, Valid: true}
	}
	if isbn10 != "" {
		nullISBN10 = sql.NullString{String: isbn10, Valid: true}
	}
	if description != "" {
		nullDescription = sql.NullString{String: description, Valid: true}
	}
	if pageCount > 0 {
		nullPageCount = sql.NullInt64{Int64: int64(pageCount), Valid: true}
	}

	result, err := database.DB.Exec(
		"INSERT INTO books (google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		googleBooksID, title, authors, nullDescription, thumbnailURL, nullISBN13, nullISBN10, nullPageCount,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetBookByISBN looks up a book by ISBN-13 or ISBN-10
func GetBookByISBN(isbn13, isbn10 string) (*Book, error) {
	var b Book

	// Try ISBN-13 first
	if isbn13 != "" {
		err := database.DB.QueryRow(
			"SELECT id, google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count, created_at FROM books WHERE isbn_13 = ?",
			isbn13,
		).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.Description, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt)
		if err == nil {
			return &b, nil
		}
	}

	// Try ISBN-10
	if isbn10 != "" {
		err := database.DB.QueryRow(
			"SELECT id, google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count, created_at FROM books WHERE isbn_10 = ?",
			isbn10,
		).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.Description, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt)
		if err == nil {
			return &b, nil
		}
	}

	return nil, sql.ErrNoRows
}

// UpdateBookISBN updates the ISBN and page count for an existing book
func UpdateBookISBN(bookID int64, isbn13, isbn10 string, pageCount int) error {
	var nullISBN13, nullISBN10 sql.NullString
	var nullPageCount sql.NullInt64

	if isbn13 != "" {
		nullISBN13 = sql.NullString{String: isbn13, Valid: true}
	}
	if isbn10 != "" {
		nullISBN10 = sql.NullString{String: isbn10, Valid: true}
	}
	if pageCount > 0 {
		nullPageCount = sql.NullInt64{Int64: int64(pageCount), Valid: true}
	}

	_, err := database.DB.Exec(
		"UPDATE books SET isbn_13 = COALESCE(isbn_13, ?), isbn_10 = COALESCE(isbn_10, ?), page_count = COALESCE(page_count, ?) WHERE id = ?",
		nullISBN13, nullISBN10, nullPageCount, bookID,
	)
	return err
}

// GetOrCreateBook creates a book or returns existing one (legacy function without ISBN support)
func GetOrCreateBook(googleBooksID, title, authors, thumbnailURL string) (*Book, error) {
	return GetOrCreateBookWithISBN(googleBooksID, title, authors, "", thumbnailURL, "", "", 0, "")
}

// GetOrCreateBookWithISBN performs ISBN-based deduplication and canonical lookup
// Flow:
// 1. Check if book exists by ISBN-13 or ISBN-10 → return existing book
// 2. Check if book exists by Google Books ID → update with ISBN if missing, return book
// 3. If book has ISBN, call Google Books API with isbn:{isbn} to get canonical edition
// 4. Use canonical data (or original if lookup fails) to create new book record
func GetOrCreateBookWithISBN(googleBooksID, title, authors, description, thumbnailURL, isbn13, isbn10 string, pageCount int, apiKey string) (*Book, error) {
	// Step 1: Check if book exists by ISBN
	if isbn13 != "" || isbn10 != "" {
		book, err := GetBookByISBN(isbn13, isbn10)
		if err == nil {
			log.Printf("[GetOrCreateBookWithISBN] Found existing book by ISBN: %s (ID: %d)", book.Title, book.ID)
			return book, nil
		}
	}

	// Step 2: Check if book exists by Google Books ID
	book, err := GetBookByGoogleID(googleBooksID)
	if err == nil {
		// Backfill: Update with ISBN data if we have it but book doesn't
		if (isbn13 != "" && !book.ISBN13.Valid) || (isbn10 != "" && !book.ISBN10.Valid) || (pageCount > 0 && !book.PageCount.Valid) {
			log.Printf("[GetOrCreateBookWithISBN] Backfilling ISBN data for existing book: %s", book.Title)
			UpdateBookISBN(book.ID, isbn13, isbn10, pageCount)
			// Refetch to get updated data
			return GetBookByID(book.ID)
		}
		return book, nil
	}

	// Step 3: If we have ISBN, try canonical lookup via Google Books API
	finalGoogleBooksID := googleBooksID
	finalTitle := title
	finalAuthors := authors
	finalDescription := description
	finalThumbnailURL := thumbnailURL
	finalISBN13 := isbn13
	finalISBN10 := isbn10
	finalPageCount := pageCount

	if (isbn13 != "" || isbn10 != "") && apiKey != "" {
		canonical, err := services.GetBookByISBN(isbn13, isbn10, apiKey)
		if err == nil && canonical != nil {
			log.Printf("[GetOrCreateBookWithISBN] Using canonical data from ISBN lookup: %s", canonical.Title)
			finalGoogleBooksID = canonical.GoogleBooksID
			finalTitle = canonical.Title
			finalAuthors = canonical.Authors
			finalDescription = canonical.Description
			finalThumbnailURL = canonical.ThumbnailURL
			finalISBN13 = canonical.ISBN13
			finalISBN10 = canonical.ISBN10
			finalPageCount = canonical.PageCount

			// Check again if this canonical book already exists
			existingBook, err := GetBookByGoogleID(finalGoogleBooksID)
			if err == nil {
				// Update with ISBN if missing
				if !existingBook.ISBN13.Valid || !existingBook.ISBN10.Valid {
					UpdateBookISBN(existingBook.ID, finalISBN13, finalISBN10, finalPageCount)
					return GetBookByID(existingBook.ID)
				}
				return existingBook, nil
			}
		}
	}

	// Step 4: Create new book record
	id, err := CreateBook(finalGoogleBooksID, finalTitle, finalAuthors, finalDescription, finalThumbnailURL, finalISBN13, finalISBN10, finalPageCount)
	if err != nil {
		return nil, err
	}

	return GetBookByID(id)
}

// DescriptionText returns the description as a string (empty if not set)
func (b Book) DescriptionText() string {
	if b.Description.Valid {
		return b.Description.String
	}
	return ""
}

// UpdateBookDescription updates a book's description
func UpdateBookDescription(bookID int64, description string) error {
	var nullDescription sql.NullString
	if description != "" {
		nullDescription = sql.NullString{String: description, Valid: true}
	}
	_, err := database.DB.Exec(
		"UPDATE books SET description = ? WHERE id = ?",
		nullDescription, bookID,
	)
	return err
}

// GetBooksWithoutDescription returns all books that don't have a description set
func GetBooksWithoutDescription() ([]Book, error) {
	rows, err := database.DB.Query(`
		SELECT id, google_books_id, title, authors, description, thumbnail_url, isbn_13, isbn_10, page_count, created_at
		FROM books
		WHERE description IS NULL OR description = ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.Description, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}
