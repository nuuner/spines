package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/spines/internal/cache"
)

// cachedImage holds the image data and content type
type cachedImage struct {
	Data        []byte
	ContentType string
}

// imageCache stores Google Books thumbnails for 1 hour
var imageCache = cache.New(1 * time.Hour)

// ProxyBookCover fetches and caches book cover images from Google Books
func ProxyBookCover(c *fiber.Ctx) error {
	bookID := c.Params("id")
	if bookID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Book ID is required")
	}

	// Optional parameters
	zoom := c.Query("zoom", "1")
	edge := c.Query("edge", "curl")

	// Build cache key from all params
	cacheKey := "book_cover:" + bookID + ":" + zoom + ":" + edge

	// Check cache first
	if cached, found := imageCache.Get(cacheKey); found {
		img := cached.(cachedImage)
		c.Set("Content-Type", img.ContentType)
		c.Set("Cache-Control", "public, max-age=3600")
		c.Set("X-Cache", "HIT")
		return c.Send(img.Data)
	}

	// Build Google Books thumbnail URL
	googleURL := "https://books.google.com/books/content?id=" + bookID +
		"&printsec=frontcover&img=1&zoom=" + zoom + "&edge=" + edge

	// Fetch from Google Books
	resp, err := http.Get(googleURL)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("Failed to fetch image")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).SendString("Image not found")
	}

	// Read the image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read image")
	}

	// Get content type from response
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	// Store in cache
	imageCache.Set(cacheKey, cachedImage{
		Data:        data,
		ContentType: contentType,
	})

	// Return the image
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("X-Cache", "MISS")
	return c.Send(data)
}
