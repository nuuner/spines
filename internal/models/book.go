package models

import (
	"time"

	"github.com/nuuner/spines/internal/database"
)

type Book struct {
	ID            int64
	GoogleBooksID string
	Title         string
	Authors       string
	ThumbnailURL  string
	CreatedAt     time.Time
}

func GetBookByGoogleID(googleBooksID string) (*Book, error) {
	var b Book
	err := database.DB.QueryRow(
		"SELECT id, google_books_id, title, authors, thumbnail_url, created_at FROM books WHERE google_books_id = ?",
		googleBooksID,
	).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.ThumbnailURL, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func GetBookByID(id int64) (*Book, error) {
	var b Book
	err := database.DB.QueryRow(
		"SELECT id, google_books_id, title, authors, thumbnail_url, created_at FROM books WHERE id = ?",
		id,
	).Scan(&b.ID, &b.GoogleBooksID, &b.Title, &b.Authors, &b.ThumbnailURL, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func CreateBook(googleBooksID, title, authors, thumbnailURL string) (int64, error) {
	result, err := database.DB.Exec(
		"INSERT INTO books (google_books_id, title, authors, thumbnail_url) VALUES (?, ?, ?, ?)",
		googleBooksID, title, authors, thumbnailURL,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func GetOrCreateBook(googleBooksID, title, authors, thumbnailURL string) (*Book, error) {
	book, err := GetBookByGoogleID(googleBooksID)
	if err == nil {
		return book, nil
	}

	id, err := CreateBook(googleBooksID, title, authors, thumbnailURL)
	if err != nil {
		return nil, err
	}

	return GetBookByID(id)
}
