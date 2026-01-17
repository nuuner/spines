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
	ImageLinks          *GoogleImageLinks    `json:"imageLinks"`
	IndustryIdentifiers []IndustryIdentifier `json:"industryIdentifiers"`
	PageCount           int                  `json:"pageCount"`
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
	ThumbnailURL  string
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
	for _, item := range result.Items {
		book := BookSearchResult{
			GoogleBooksID: item.ID,
			Title:         item.VolumeInfo.Title,
			Authors:       strings.Join(item.VolumeInfo.Authors, ", "),
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

