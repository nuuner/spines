package models

import (
	"database/sql"
	"time"

	"github.com/nuuner/spines/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int64
	Username       string
	DisplayName    string
	Description    string
	PasswordHash   sql.NullString
	ProfilePicture sql.NullString
	CreatedAt      time.Time
}

// HasPassword returns true if the user has a password set
func (u *User) HasPassword() bool {
	return u.PasswordHash.Valid && u.PasswordHash.String != ""
}

// CheckPassword verifies the password against the stored hash
func (u *User) CheckPassword(password string) bool {
	if !u.HasPassword() {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash.String), []byte(password))
	return err == nil
}

// GetProfilePictureURL returns the URL for the user's profile picture or default
func (u *User) GetProfilePictureURL() string {
	if u.ProfilePicture.Valid && u.ProfilePicture.String != "" {
		return "/static/uploads/avatars/" + u.ProfilePicture.String
	}
	return "/static/uploads/avatars/default.svg"
}

// HasProfilePicture returns true if the user has a custom profile picture
func (u *User) HasProfilePicture() bool {
	return u.ProfilePicture.Valid && u.ProfilePicture.String != ""
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func GetAllUsers() ([]User, error) {
	rows, err := database.DB.Query("SELECT id, username, display_name, description, password_hash, profile_picture, created_at FROM users ORDER BY display_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func GetUserByID(id int64) (*User, error) {
	var u User
	err := database.DB.QueryRow(
		"SELECT id, username, display_name, description, password_hash, profile_picture, created_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByUsername(username string) (*User, error) {
	var u User
	err := database.DB.QueryRow(
		"SELECT id, username, display_name, description, password_hash, profile_picture, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Description, &u.PasswordHash, &u.ProfilePicture, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateUser(username, displayName, description string) (int64, error) {
	result, err := database.DB.Exec(
		"INSERT INTO users (username, display_name, description) VALUES (?, ?, ?)",
		username, displayName, description,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateUser(id int64, username, displayName, description string) error {
	_, err := database.DB.Exec(
		"UPDATE users SET username = ?, display_name = ?, description = ? WHERE id = ?",
		username, displayName, description, id,
	)
	return err
}

func DeleteUser(id int64) error {
	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func CountUsers() (int, error) {
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// SetUserPassword sets a password for the user
func SetUserPassword(userID int64, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec(
		"UPDATE users SET password_hash = ? WHERE id = ?",
		hash, userID,
	)
	return err
}

// ClearUserPassword removes the user's password (disables login)
func ClearUserPassword(userID int64) error {
	_, err := database.DB.Exec(
		"UPDATE users SET password_hash = NULL WHERE id = ?",
		userID,
	)
	return err
}

// UpdateUserProfile updates only display_name and description
func UpdateUserProfile(userID int64, displayName, description string) error {
	_, err := database.DB.Exec(
		"UPDATE users SET display_name = ?, description = ? WHERE id = ?",
		displayName, description, userID,
	)
	return err
}

// UpdateUserProfilePicture sets the profile picture filename for a user
func UpdateUserProfilePicture(userID int64, filename string) error {
	_, err := database.DB.Exec(
		"UPDATE users SET profile_picture = ? WHERE id = ?",
		filename, userID,
	)
	return err
}

// ClearUserProfilePicture removes the profile picture for a user
func ClearUserProfilePicture(userID int64) error {
	_, err := database.DB.Exec(
		"UPDATE users SET profile_picture = NULL WHERE id = ?",
		userID,
	)
	return err
}
