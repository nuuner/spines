package handlers

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nuuner/spines/internal/models"
	_ "golang.org/x/image/webp"
)

const (
	avatarDir     = "./web/static/uploads/avatars"
	maxAvatarSize = 2 * 1024 * 1024 // 2MB
	avatarSize    = 200             // 200x200 pixels
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// UploadAvatar handles profile picture uploads
func UploadAvatar(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	// Get the uploaded file
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Redirect("/profile?error=No+file+uploaded")
	}

	// Check file size
	if file.Size > maxAvatarSize {
		return c.Redirect("/profile?error=File+too+large.+Maximum+size+is+2MB")
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+read+file")
	}
	defer src.Close()

	// Read the first 512 bytes to detect content type
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+read+file")
	}

	// Detect MIME type
	mimeType := http.DetectContentType(buffer[:n])
	if !allowedMimeTypes[mimeType] {
		return c.Redirect("/profile?error=Invalid+file+type.+Allowed:+JPEG,+PNG,+GIF,+WebP")
	}

	// Reset file reader to beginning
	src.Seek(0, 0)

	// Decode the image
	img, _, err := image.Decode(src)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+process+image")
	}

	// Crop to square (center crop) and resize to avatarSize x avatarSize
	img = imaging.Fill(img, avatarSize, avatarSize, imaging.Center, imaging.Lanczos)

	// Generate UUID filename
	filename := uuid.New().String() + ".webp"

	// Ensure directory exists
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return c.Redirect("/profile?error=Failed+to+save+image")
	}

	// Save as WebP
	destPath := filepath.Join(avatarDir, filename)
	outFile, err := os.Create(destPath)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+save+image")
	}
	defer outFile.Close()

	err = webp.Encode(outFile, img, &webp.Options{Quality: 85})
	if err != nil {
		os.Remove(destPath)
		return c.Redirect("/profile?error=Failed+to+save+image")
	}

	// Delete old avatar if exists
	if user.HasProfilePicture() {
		oldPath := filepath.Join(avatarDir, user.ProfilePicture.String)
		os.Remove(oldPath) // Ignore errors for old file deletion
	}

	// Update database
	err = models.UpdateUserProfilePicture(user.ID, filename)
	if err != nil {
		// Try to clean up the newly uploaded file
		os.Remove(destPath)
		return c.Redirect("/profile?error=Failed+to+update+profile")
	}

	return c.Redirect("/profile?success=Profile+picture+updated")
}

// RemoveAvatar handles profile picture removal
func RemoveAvatar(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)

	// Delete file if exists
	if user.HasProfilePicture() {
		oldPath := filepath.Join(avatarDir, user.ProfilePicture.String)
		os.Remove(oldPath) // Ignore errors
	}

	// Clear database
	err := models.ClearUserProfilePicture(user.ID)
	if err != nil {
		return c.Redirect("/profile?error=Failed+to+remove+profile+picture")
	}

	return c.Redirect("/profile?success=Profile+picture+removed")
}
