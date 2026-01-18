package models

import (
	"database/sql"
	"strings"
	"time"

	"github.com/nuuner/spines/internal/database"
)

type UserBook struct {
	ID                int64
	UserID            int64
	BookID            int64
	Shelf             string
	SubStatus         sql.NullString
	AddedAt           sql.NullString
	StartedReadingAt  sql.NullString
	FinishedReadingAt sql.NullString
	Book              *Book
}

// parseDateTime parses a SQLite datetime string into time.Time
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}

type ShelfBooks struct {
	WantToRead       []UserBook
	CurrentlyReading []UserBook
	Read             []UserBook
}

func GetUserBooks(userID int64) (*ShelfBooks, error) {
	rows, err := database.DB.Query(`
		SELECT ub.id, ub.user_id, ub.book_id, ub.shelf, ub.sub_status,
		       ub.added_at, ub.started_reading_at, ub.finished_reading_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM user_books ub
		JOIN books b ON ub.book_id = b.id
		WHERE ub.user_id = ?
		ORDER BY COALESCE(ub.added_at, '1970-01-01') DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shelves := &ShelfBooks{}
	for rows.Next() {
		var ub UserBook
		var b Book
		if err := rows.Scan(
			&ub.ID, &ub.UserID, &ub.BookID, &ub.Shelf, &ub.SubStatus,
			&ub.AddedAt, &ub.StartedReadingAt, &ub.FinishedReadingAt,
			&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		ub.Book = &b

		switch ub.Shelf {
		case "want_to_read":
			shelves.WantToRead = append(shelves.WantToRead, ub)
		case "currently_reading":
			shelves.CurrentlyReading = append(shelves.CurrentlyReading, ub)
		case "read":
			shelves.Read = append(shelves.Read, ub)
		}
	}
	return shelves, rows.Err()
}

// GetRandomCurrentlyReadingByUserIDs returns a map of userID -> random currently reading UserBook
func GetRandomCurrentlyReadingByUserIDs(userIDs []int64) (map[int64]*UserBook, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*UserBook), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT ub.id, ub.user_id, ub.book_id, ub.shelf, ub.sub_status,
		       ub.added_at, ub.started_reading_at, ub.finished_reading_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM user_books ub
		JOIN books b ON ub.book_id = b.id
		WHERE ub.shelf = 'currently_reading' AND ub.user_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY ub.user_id, RANDOM()
	`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]*UserBook)
	for rows.Next() {
		var ub UserBook
		var b Book
		if err := rows.Scan(
			&ub.ID, &ub.UserID, &ub.BookID, &ub.Shelf, &ub.SubStatus,
			&ub.AddedAt, &ub.StartedReadingAt, &ub.FinishedReadingAt,
			&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		ub.Book = &b
		// Only keep the first (random) book per user
		if _, exists := result[ub.UserID]; !exists {
			result[ub.UserID] = &ub
		}
	}
	return result, rows.Err()
}

// GetShelfBooksPaginated returns books for a specific shelf with pagination
// Returns the books slice, total count for that shelf, and any error
func GetShelfBooksPaginated(userID int64, shelf string, offset, limit int) ([]UserBook, int, error) {
	// Get total count for this shelf
	var total int
	err := database.DB.QueryRow(`
		SELECT COUNT(*) FROM user_books WHERE user_id = ? AND shelf = ?
	`, userID, shelf).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated books
	rows, err := database.DB.Query(`
		SELECT ub.id, ub.user_id, ub.book_id, ub.shelf, ub.sub_status,
		       ub.added_at, ub.started_reading_at, ub.finished_reading_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM user_books ub
		JOIN books b ON ub.book_id = b.id
		WHERE ub.user_id = ? AND ub.shelf = ?
		ORDER BY COALESCE(ub.added_at, '1970-01-01') DESC
		LIMIT ? OFFSET ?
	`, userID, shelf, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var books []UserBook
	for rows.Next() {
		var ub UserBook
		var b Book
		if err := rows.Scan(
			&ub.ID, &ub.UserID, &ub.BookID, &ub.Shelf, &ub.SubStatus,
			&ub.AddedAt, &ub.StartedReadingAt, &ub.FinishedReadingAt,
			&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.ThumbnailURL, &b.ISBN13, &b.ISBN10, &b.PageCount, &b.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		ub.Book = &b
		books = append(books, ub)
	}
	return books, total, rows.Err()
}

func GetUserBook(userID, bookID int64) (*UserBook, error) {
	var ub UserBook
	err := database.DB.QueryRow(`
		SELECT ub.id, ub.user_id, ub.book_id, ub.shelf, ub.sub_status,
		       ub.added_at, ub.started_reading_at, ub.finished_reading_at
		FROM user_books ub
		WHERE ub.user_id = ? AND ub.book_id = ?
	`, userID, bookID).Scan(&ub.ID, &ub.UserID, &ub.BookID, &ub.Shelf, &ub.SubStatus,
		&ub.AddedAt, &ub.StartedReadingAt, &ub.FinishedReadingAt)
	if err != nil {
		return nil, err
	}
	return &ub, nil
}

func AddBookToShelf(userID, bookID int64, shelf string, subStatus sql.NullString) error {
	var query string

	switch shelf {
	case "currently_reading":
		query = `INSERT INTO user_books (user_id, book_id, shelf, sub_status, added_at, started_reading_at)
		         VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	case "read":
		query = `INSERT INTO user_books (user_id, book_id, shelf, sub_status, added_at, started_reading_at, finished_reading_at)
		         VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	default:
		query = `INSERT INTO user_books (user_id, book_id, shelf, sub_status, added_at)
		         VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
	}

	_, err := database.DB.Exec(query, userID, bookID, shelf, subStatus)
	return err
}

func UpdateUserBook(userID, bookID int64, shelf string, subStatus sql.NullString) error {
	var query string

	switch shelf {
	case "want_to_read":
		// Moving backward: clear both timestamps
		query = `UPDATE user_books
		         SET shelf = ?, sub_status = ?, started_reading_at = NULL, finished_reading_at = NULL
		         WHERE user_id = ? AND book_id = ?`
	case "currently_reading":
		// If finished_reading_at is set (re-read scenario), start fresh with new started_reading_at
		// Otherwise, preserve existing started_reading_at or set it if NULL
		query = `UPDATE user_books
		         SET shelf = ?, sub_status = ?,
		             started_reading_at = CASE
		                 WHEN finished_reading_at IS NOT NULL THEN CURRENT_TIMESTAMP
		                 ELSE COALESCE(started_reading_at, CURRENT_TIMESTAMP)
		             END,
		             finished_reading_at = NULL
		         WHERE user_id = ? AND book_id = ?`
	case "read":
		// Set finished_reading_at, preserve or set started_reading_at
		query = `UPDATE user_books
		         SET shelf = ?, sub_status = ?,
		             started_reading_at = COALESCE(started_reading_at, CURRENT_TIMESTAMP),
		             finished_reading_at = CURRENT_TIMESTAMP
		         WHERE user_id = ? AND book_id = ?`
	default:
		query = `UPDATE user_books SET shelf = ?, sub_status = ? WHERE user_id = ? AND book_id = ?`
	}

	_, err := database.DB.Exec(query, shelf, subStatus, userID, bookID)
	return err
}

func RemoveBookFromShelf(userID, bookID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM user_books WHERE user_id = ? AND book_id = ?",
		userID, bookID,
	)
	return err
}

// SubStatusDisplay returns a human-readable version of the sub_status
func (ub UserBook) SubStatusDisplay() string {
	if !ub.SubStatus.Valid {
		return ""
	}
	switch ub.SubStatus.String {
	case "just_started":
		return "Just started"
	case "25_percent":
		return "25%"
	case "50_percent":
		return "50%"
	case "75_percent":
		return "75%"
	case "almost_finished":
		return "Almost finished"
	case "need_to_buy":
		return "Do not own"
	case "already_own":
		return "Already own"
	default:
		return ub.SubStatus.String
	}
}

// ReadingProgress returns the progress percentage (0-100) for currently reading books
// Returns -1 if not a percentage-based status
func (ub UserBook) ReadingProgress() int {
	if !ub.SubStatus.Valid {
		return -1
	}
	switch ub.SubStatus.String {
	case "just_started":
		return 5
	case "25_percent":
		return 25
	case "50_percent":
		return 50
	case "75_percent":
		return 75
	case "almost_finished":
		return 95
	default:
		return -1
	}
}

// UpdateUserBookDates updates only the date fields for a user's book
func UpdateUserBookDates(userID, bookID int64, addedAt, startedReadingAt, finishedReadingAt sql.NullString) error {
	_, err := database.DB.Exec(`
		UPDATE user_books
		SET added_at = ?, started_reading_at = ?, finished_reading_at = ?
		WHERE user_id = ? AND book_id = ?`,
		addedAt, startedReadingAt, finishedReadingAt, userID, bookID)
	return err
}

// AddedAtFormatted returns the added_at date formatted for HTML datetime-local input
func (ub UserBook) AddedAtFormatted() string {
	if !ub.AddedAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.AddedAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

// StartedReadingAtFormatted returns the started_reading_at date formatted for HTML datetime-local input
func (ub UserBook) StartedReadingAtFormatted() string {
	if !ub.StartedReadingAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.StartedReadingAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

// FinishedReadingAtFormatted returns the finished_reading_at date formatted for HTML datetime-local input
func (ub UserBook) FinishedReadingAtFormatted() string {
	if !ub.FinishedReadingAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.FinishedReadingAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

// AddedAtDisplay returns the added_at date formatted for display (e.g., "Jan 15, 2026")
func (ub UserBook) AddedAtDisplay() string {
	if !ub.AddedAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.AddedAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

// StartedReadingAtDisplay returns the started_reading_at date formatted for display (e.g., "Jan 15, 2026")
func (ub UserBook) StartedReadingAtDisplay() string {
	if !ub.StartedReadingAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.StartedReadingAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}

// FinishedReadingAtDisplay returns the finished_reading_at date formatted for display (e.g., "Jan 15, 2026")
func (ub UserBook) FinishedReadingAtDisplay() string {
	if !ub.FinishedReadingAt.Valid {
		return ""
	}
	t, err := parseDateTime(ub.FinishedReadingAt.String)
	if err != nil || t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006")
}
