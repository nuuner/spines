package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nuuner/spines/internal/database"
)

// Event types
const (
	EventBookAdded       = "book_added"
	EventBookMoved       = "book_moved"
	EventReadingProgress = "reading_progress"
	EventBookRemoved     = "book_removed"
)

// Event represents a user activity event
type Event struct {
	ID        int64
	UserID    int64
	EventType string
	BookID    sql.NullInt64
	Shelf     sql.NullString
	OldValue  sql.NullString
	NewValue  sql.NullString
	CreatedAt time.Time
	// Joined data
	User *User
	Book *Book
}

// CreateEvent creates a new event record
func CreateEvent(userID int64, eventType string, bookID sql.NullInt64, shelf, oldValue, newValue sql.NullString) error {
	_, err := database.DB.Exec(`
		INSERT INTO events (user_id, event_type, book_id, shelf, old_value, new_value)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, eventType, bookID, shelf, oldValue, newValue)
	return err
}

// CreateBookAddedEvent creates an event for when a user adds a book to a shelf
func CreateBookAddedEvent(userID, bookID int64, shelf string) error {
	return CreateEvent(
		userID,
		EventBookAdded,
		sql.NullInt64{Int64: bookID, Valid: true},
		sql.NullString{String: shelf, Valid: true},
		sql.NullString{},
		sql.NullString{},
	)
}

// CreateBookMovedEvent creates an event for when a user moves a book to a different shelf
func CreateBookMovedEvent(userID, bookID int64, oldShelf, newShelf string) error {
	return CreateEvent(
		userID,
		EventBookMoved,
		sql.NullInt64{Int64: bookID, Valid: true},
		sql.NullString{String: newShelf, Valid: true},
		sql.NullString{String: oldShelf, Valid: true},
		sql.NullString{String: newShelf, Valid: true},
	)
}

// CreateReadingProgressEvent creates an event for when a user updates their reading progress
func CreateReadingProgressEvent(userID, bookID int64, oldProgress, newProgress string) error {
	return CreateEvent(
		userID,
		EventReadingProgress,
		sql.NullInt64{Int64: bookID, Valid: true},
		sql.NullString{String: "currently_reading", Valid: true},
		sql.NullString{String: oldProgress, Valid: oldProgress != ""},
		sql.NullString{String: newProgress, Valid: newProgress != ""},
	)
}

// CreateBookRemovedEvent creates an event for when a user removes a book from their shelf
func CreateBookRemovedEvent(userID, bookID int64, shelf string) error {
	return CreateEvent(
		userID,
		EventBookRemoved,
		sql.NullInt64{Int64: bookID, Valid: true},
		sql.NullString{String: shelf, Valid: true},
		sql.NullString{},
		sql.NullString{},
	)
}

// GetLatestEventPerUser returns the most recent event for each user
// This is useful for showing a news feed of recent activity
func GetLatestEventPerUser(limit int) ([]Event, error) {
	rows, err := database.DB.Query(`
		SELECT e.id, e.user_id, e.event_type, e.book_id, e.shelf, e.old_value, e.new_value, e.created_at,
		       u.id, u.username, u.display_name, u.description, u.password_hash, u.profile_picture, COALESCE(u.theme, 'light'), u.created_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM events e
		INNER JOIN (
			SELECT user_id, MAX(id) as max_id
			FROM events
			GROUP BY user_id
		) latest ON e.id = latest.max_id
		INNER JOIN users u ON e.user_id = u.id
		LEFT JOIN books b ON e.book_id = b.id
		ORDER BY e.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		var u User
		var b Book
		var bookID sql.NullInt64
		var bookGoogleID, bookTitle, bookAuthors, bookThumbnail, bookCreatedAt sql.NullString

		if err := rows.Scan(
			&ev.ID, &ev.UserID, &ev.EventType, &ev.BookID, &ev.Shelf, &ev.OldValue, &ev.NewValue, &ev.CreatedAt,
			&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.Theme, &u.CreatedAt,
			&bookID, &bookGoogleID, &bookTitle, &bookAuthors, &bookThumbnail, &b.ISBN13, &b.ISBN10, &b.PageCount, &bookCreatedAt,
		); err != nil {
			return nil, err
		}

		ev.User = &u

		// Only set book if it exists
		if bookID.Valid {
			b.ID = bookID.Int64
			b.GoogleBooksID = bookGoogleID.String
			b.Title = bookTitle.String
			b.Authors = bookAuthors.String
			b.ThumbnailURL = bookThumbnail.String
			if bookCreatedAt.Valid {
				if t, err := time.Parse("2006-01-02 15:04:05", bookCreatedAt.String); err == nil {
					b.CreatedAt = t
				}
			}
			ev.Book = &b
		}

		events = append(events, ev)
	}
	return events, rows.Err()
}

// GetRecentEvents returns the most recent events across all users
func GetRecentEvents(limit int) ([]Event, error) {
	rows, err := database.DB.Query(`
		SELECT e.id, e.user_id, e.event_type, e.book_id, e.shelf, e.old_value, e.new_value, e.created_at,
		       u.id, u.username, u.display_name, u.description, u.password_hash, u.profile_picture, COALESCE(u.theme, 'light'), u.created_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM events e
		INNER JOIN users u ON e.user_id = u.id
		LEFT JOIN books b ON e.book_id = b.id
		ORDER BY e.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetUserEvents returns events for a specific user
func GetUserEvents(userID int64, limit int) ([]Event, error) {
	rows, err := database.DB.Query(`
		SELECT e.id, e.user_id, e.event_type, e.book_id, e.shelf, e.old_value, e.new_value, e.created_at,
		       u.id, u.username, u.display_name, u.description, u.password_hash, u.profile_picture, COALESCE(u.theme, 'light'), u.created_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM events e
		INNER JOIN users u ON e.user_id = u.id
		LEFT JOIN books b ON e.book_id = b.id
		WHERE e.user_id = ?
		ORDER BY e.created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetLatestNEventsPerUser returns the most recent N events per user, sorted by date
// This is useful for the dashboard news feed showing multiple events per user
func GetLatestNEventsPerUser(eventsPerUser, totalLimit int) ([]Event, error) {
	// Use a window function to get the latest N events per user
	rows, err := database.DB.Query(`
		SELECT e.id, e.user_id, e.event_type, e.book_id, e.shelf, e.old_value, e.new_value, e.created_at,
		       u.id, u.username, u.display_name, u.description, u.password_hash, u.profile_picture, COALESCE(u.theme, 'light'), u.created_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at DESC) as rn
			FROM events
		) e
		INNER JOIN users u ON e.user_id = u.id
		LEFT JOIN books b ON e.book_id = b.id
		WHERE e.rn <= ?
		ORDER BY e.created_at DESC
		LIMIT ?
	`, eventsPerUser, totalLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetLatestEventPerUserByIDs returns the most recent event for each specified user
func GetLatestEventPerUserByIDs(userIDs []int64) (map[int64]*Event, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*Event), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT e.id, e.user_id, e.event_type, e.book_id, e.shelf, e.old_value, e.new_value, e.created_at,
		       u.id, u.username, u.display_name, u.description, u.password_hash, u.profile_picture, COALESCE(u.theme, 'light'), u.created_at,
		       b.id, b.google_books_id, b.title, b.authors, b.thumbnail_url, b.isbn_13, b.isbn_10, b.page_count, b.created_at
		FROM events e
		INNER JOIN (
			SELECT user_id, MAX(id) as max_id
			FROM events
			WHERE user_id IN (` + strings.Join(placeholders, ",") + `)
			GROUP BY user_id
		) latest ON e.id = latest.max_id
		INNER JOIN users u ON e.user_id = u.id
		LEFT JOIN books b ON e.book_id = b.id
	`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]*Event)
	for rows.Next() {
		var ev Event
		var u User
		var b Book
		var bookID sql.NullInt64
		var bookGoogleID, bookTitle, bookAuthors, bookThumbnail, bookCreatedAt sql.NullString

		if err := rows.Scan(
			&ev.ID, &ev.UserID, &ev.EventType, &ev.BookID, &ev.Shelf, &ev.OldValue, &ev.NewValue, &ev.CreatedAt,
			&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.Theme, &u.CreatedAt,
			&bookID, &bookGoogleID, &bookTitle, &bookAuthors, &bookThumbnail, &b.ISBN13, &b.ISBN10, &b.PageCount, &bookCreatedAt,
		); err != nil {
			return nil, err
		}

		ev.User = &u

		if bookID.Valid {
			b.ID = bookID.Int64
			b.GoogleBooksID = bookGoogleID.String
			b.Title = bookTitle.String
			b.Authors = bookAuthors.String
			b.ThumbnailURL = bookThumbnail.String
			if bookCreatedAt.Valid {
				if t, err := time.Parse("2006-01-02 15:04:05", bookCreatedAt.String); err == nil {
					b.CreatedAt = t
				}
			}
			ev.Book = &b
		}

		result[ev.UserID] = &ev
	}
	return result, rows.Err()
}

// scanEvents is a helper to scan event rows with user and book data
func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var ev Event
		var u User
		var b Book
		var bookID sql.NullInt64
		var bookGoogleID, bookTitle, bookAuthors, bookThumbnail, bookCreatedAt sql.NullString

		if err := rows.Scan(
			&ev.ID, &ev.UserID, &ev.EventType, &ev.BookID, &ev.Shelf, &ev.OldValue, &ev.NewValue, &ev.CreatedAt,
			&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.Theme, &u.CreatedAt,
			&bookID, &bookGoogleID, &bookTitle, &bookAuthors, &bookThumbnail, &b.ISBN13, &b.ISBN10, &b.PageCount, &bookCreatedAt,
		); err != nil {
			return nil, err
		}

		ev.User = &u

		if bookID.Valid {
			b.ID = bookID.Int64
			b.GoogleBooksID = bookGoogleID.String
			b.Title = bookTitle.String
			b.Authors = bookAuthors.String
			b.ThumbnailURL = bookThumbnail.String
			if bookCreatedAt.Valid {
				if t, err := time.Parse("2006-01-02 15:04:05", bookCreatedAt.String); err == nil {
					b.CreatedAt = t
				}
			}
			ev.Book = &b
		}

		events = append(events, ev)
	}
	return events, rows.Err()
}

// ShelfDisplay returns a human-readable version of the shelf name
func (e Event) ShelfDisplay() string {
	if !e.Shelf.Valid {
		return ""
	}
	switch e.Shelf.String {
	case "want_to_read":
		return "Want to Read"
	case "currently_reading":
		return "Currently Reading"
	case "read":
		return "Read"
	default:
		return e.Shelf.String
	}
}

// OldValueDisplay returns a human-readable version of old_value (for shelf names or progress)
func (e Event) OldValueDisplay() string {
	if !e.OldValue.Valid {
		return ""
	}
	return formatValueDisplay(e.OldValue.String)
}

// NewValueDisplay returns a human-readable version of new_value (for shelf names or progress)
func (e Event) NewValueDisplay() string {
	if !e.NewValue.Valid {
		return ""
	}
	return formatValueDisplay(e.NewValue.String)
}

func formatValueDisplay(val string) string {
	switch val {
	case "want_to_read":
		return "Want to Read"
	case "currently_reading":
		return "Currently Reading"
	case "read":
		return "Read"
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
	default:
		return val
	}
}

// EventDescription returns a human-readable description of the event
func (e Event) EventDescription() string {
	bookTitle := ""
	if e.Book != nil {
		bookTitle = e.Book.Title
	}

	switch e.EventType {
	case EventBookAdded:
		return "added \"" + bookTitle + "\" to " + e.ShelfDisplay()
	case EventBookMoved:
		return "moved \"" + bookTitle + "\" to " + e.NewValueDisplay()
	case EventReadingProgress:
		if e.NewValue.Valid {
			return "is " + e.NewValueDisplay() + " through \"" + bookTitle + "\""
		}
		return "updated progress on \"" + bookTitle + "\""
	case EventBookRemoved:
		return "removed \"" + bookTitle + "\" from " + e.ShelfDisplay()
	default:
		return "performed an action"
	}
}

// TimeAgo returns a human-readable relative time string
func (e Event) TimeAgo() string {
	now := time.Now()
	diff := now.Sub(e.CreatedAt)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return e.CreatedAt.Format("Jan 2, 2006")
	}
}
