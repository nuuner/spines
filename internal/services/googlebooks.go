package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nuuner/spines/internal/cache"
)

// searchCache stores Google Books API search results for 8 hours
var searchCache = cache.New(8 * time.Hour)

type GoogleBooksResponse struct {
	Items []GoogleBookItem `json:"items"`
}

type GoogleBookItem struct {
	ID         string           `json:"id"`
	VolumeInfo GoogleVolumeInfo `json:"volumeInfo"`
}

type GoogleVolumeInfo struct {
	Title               string               `json:"title"`
	Authors             []string             `json:"authors"`
	Description         string               `json:"description"`
	ImageLinks          *GoogleImageLinks    `json:"imageLinks"`
	IndustryIdentifiers []IndustryIdentifier `json:"industryIdentifiers"`
	PageCount           int                  `json:"pageCount"`
	PublishedDate       string               `json:"publishedDate"`
	Language            string               `json:"language"`
}

type IndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type GoogleImageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
}

type BookSearchResult struct {
	GoogleBooksID string
	Title         string
	Authors       string
	Description   string
	ThumbnailURL  string
	ISBN13        string
	ISBN10        string
	PageCount     int
	PublishedYear string
	Language      string
}

func SearchBooks(query string, apiKey string) ([]BookSearchResult, error) {
	// Check cache first
	if cached, found := searchCache.Get(query); found {
		log.Printf("[Cache HIT] %s", query)
		return cached.([]BookSearchResult), nil
	}
	log.Printf("[Cache MISS] %s", query)

	baseURL := "https://www.googleapis.com/books/v1/volumes"
	params := url.Values{}
	params.Set("q", "intitle:"+query)
	params.Set("maxResults", "40") // Request more to account for filtering
	params.Set("printType", "books") // Only return books, not magazines

	if apiKey != "" {
		params.Set("key", apiKey)
	}

	resp, err := http.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books API returned status %d", resp.StatusCode)
	}

	var result GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var books []BookSearchResult
	seenISBN13 := make(map[string]bool)
	seenISBN10 := make(map[string]bool)

	for _, item := range result.Items {
		book := BookSearchResult{
			GoogleBooksID: item.ID,
			Title:         item.VolumeInfo.Title,
			Description:   item.VolumeInfo.Description,
			PageCount:     item.VolumeInfo.PageCount,
			Language:      item.VolumeInfo.Language,
		}

		// Extract year from publishedDate (formats: "2021", "2021-05", "2021-05-04")
		if len(item.VolumeInfo.PublishedDate) >= 4 {
			book.PublishedYear = item.VolumeInfo.PublishedDate[:4]
		}

		// Defensive nil check for authors
		if item.VolumeInfo.Authors != nil {
			book.Authors = strings.Join(item.VolumeInfo.Authors, ", ")
		}

		// Extract ISBNs from IndustryIdentifiers
		for _, identifier := range item.VolumeInfo.IndustryIdentifiers {
			switch identifier.Type {
			case "ISBN_13":
				book.ISBN13 = identifier.Identifier
			case "ISBN_10":
				book.ISBN10 = identifier.Identifier
			}
		}

		// Deduplicate by ISBN - skip if we've seen this ISBN before
		isDuplicate := false
		if book.ISBN13 != "" {
			if seenISBN13[book.ISBN13] {
				isDuplicate = true
			} else {
				seenISBN13[book.ISBN13] = true
			}
		}
		if !isDuplicate && book.ISBN10 != "" {
			if seenISBN10[book.ISBN10] {
				isDuplicate = true
			} else {
				seenISBN10[book.ISBN10] = true
			}
		}
		if isDuplicate {
			continue
		}

		if item.VolumeInfo.ImageLinks != nil {
			if item.VolumeInfo.ImageLinks.Thumbnail != "" {
				book.ThumbnailURL = strings.Replace(item.VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
			} else if item.VolumeInfo.ImageLinks.SmallThumbnail != "" {
				book.ThumbnailURL = strings.Replace(item.VolumeInfo.ImageLinks.SmallThumbnail, "http://", "https://", 1)
			}
		}

		books = append(books, book)

		// Limit to 20 results after filtering
		if len(books) >= 20 {
			break
		}
	}

	// Store in cache before returning
	searchCache.Set(query, books)

	return books, nil
}

// GetBookByISBN fetches a book from Google Books API using ISBN for canonical lookup.
// Tries ISBN-13 first, then falls back to ISBN-10 if needed.
// Returns nil if no book is found.
func GetBookByISBN(isbn13, isbn10, apiKey string) (*BookSearchResult, error) {
	// Try ISBN-13 first, then ISBN-10
	isbns := []string{isbn13, isbn10}
	for _, isbn := range isbns {
		if isbn == "" {
			continue
		}

		baseURL := "https://www.googleapis.com/books/v1/volumes"
		params := url.Values{}
		params.Set("q", "isbn:"+isbn)
		params.Set("maxResults", "1")
		params.Set("printType", "books")

		if apiKey != "" {
			params.Set("key", apiKey)
		}

		resp, err := http.Get(baseURL + "?" + params.Encode())
		if err != nil {
			log.Printf("[GetBookByISBN] HTTP error for ISBN %s: %v", isbn, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[GetBookByISBN] API returned status %d for ISBN %s", resp.StatusCode, isbn)
			continue
		}

		var result GoogleBooksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("[GetBookByISBN] JSON decode error for ISBN %s: %v", isbn, err)
			continue
		}

		if len(result.Items) == 0 {
			continue
		}

		item := result.Items[0]
		book := &BookSearchResult{
			GoogleBooksID: item.ID,
			Title:         item.VolumeInfo.Title,
			Description:   item.VolumeInfo.Description,
			PageCount:     item.VolumeInfo.PageCount,
			Language:      item.VolumeInfo.Language,
		}

		// Extract year from publishedDate
		if len(item.VolumeInfo.PublishedDate) >= 4 {
			book.PublishedYear = item.VolumeInfo.PublishedDate[:4]
		}

		// Defensive nil check for authors
		if item.VolumeInfo.Authors != nil {
			book.Authors = strings.Join(item.VolumeInfo.Authors, ", ")
		}

		// Extract ISBNs from IndustryIdentifiers
		for _, identifier := range item.VolumeInfo.IndustryIdentifiers {
			switch identifier.Type {
			case "ISBN_13":
				book.ISBN13 = identifier.Identifier
			case "ISBN_10":
				book.ISBN10 = identifier.Identifier
			}
		}

		if item.VolumeInfo.ImageLinks != nil {
			if item.VolumeInfo.ImageLinks.Thumbnail != "" {
				book.ThumbnailURL = strings.Replace(item.VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
			} else if item.VolumeInfo.ImageLinks.SmallThumbnail != "" {
				book.ThumbnailURL = strings.Replace(item.VolumeInfo.ImageLinks.SmallThumbnail, "http://", "https://", 1)
			}
		}

		log.Printf("[GetBookByISBN] Found canonical book for ISBN %s: %s", isbn, book.Title)
		return book, nil
	}

	return nil, nil
}

// GetBookByGoogleBooksID fetches a book directly by its Google Books volume ID.
// Returns the description and other details for backfilling existing records.
func GetBookByGoogleBooksID(googleBooksID, apiKey string) (*BookSearchResult, error) {
	if googleBooksID == "" {
		return nil, fmt.Errorf("googleBooksID is required")
	}

	volumeURL := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes/%s", googleBooksID)
	if apiKey != "" {
		volumeURL += "?key=" + apiKey
	}

	resp, err := http.Get(volumeURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var item GoogleBookItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("JSON decode error: %v", err)
	}

	book := &BookSearchResult{
		GoogleBooksID: item.ID,
		Title:         item.VolumeInfo.Title,
		Description:   item.VolumeInfo.Description,
		PageCount:     item.VolumeInfo.PageCount,
		Language:      item.VolumeInfo.Language,
	}

	if item.VolumeInfo.Authors != nil {
		book.Authors = strings.Join(item.VolumeInfo.Authors, ", ")
	}

	for _, identifier := range item.VolumeInfo.IndustryIdentifiers {
		switch identifier.Type {
		case "ISBN_13":
			book.ISBN13 = identifier.Identifier
		case "ISBN_10":
			book.ISBN10 = identifier.Identifier
		}
	}

	if item.VolumeInfo.ImageLinks != nil {
		if item.VolumeInfo.ImageLinks.Thumbnail != "" {
			book.ThumbnailURL = strings.Replace(item.VolumeInfo.ImageLinks.Thumbnail, "http://", "https://", 1)
		}
	}

	return book, nil
}

