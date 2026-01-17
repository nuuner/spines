package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/nuuner/spines/internal/database"
)

const (
	SessionTypeAdmin = "admin"
	SessionTypeUser  = "user"
)

type Session struct {
	ID          int64
	Token       string
	UserID      sql.NullInt64
	SessionType string
	ExpiresAt   time.Time
}

// CreateAdminSession creates a session for admin authentication
func CreateAdminSession() (*Session, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	result, err := database.DB.Exec(
		"INSERT INTO sessions (token, expires_at, session_type) VALUES (?, ?, ?)",
		token, expiresAt, SessionTypeAdmin,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:          id,
		Token:       token,
		SessionType: SessionTypeAdmin,
		ExpiresAt:   expiresAt,
	}, nil
}

// CreateUserSession creates a session for a specific user
func CreateUserSession(userID int64) (*Session, error) {
	token := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	result, err := database.DB.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, session_type) VALUES (?, ?, ?, ?)",
		token, userID, expiresAt, SessionTypeUser,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:          id,
		Token:       token,
		UserID:      sql.NullInt64{Int64: userID, Valid: true},
		SessionType: SessionTypeUser,
		ExpiresAt:   expiresAt,
	}, nil
}

// CreateSession creates an admin session (for backwards compatibility)
func CreateSession() (*Session, error) {
	return CreateAdminSession()
}

func GetSessionByToken(token string) (*Session, error) {
	var s Session
	err := database.DB.QueryRow(
		"SELECT id, token, user_id, session_type, expires_at FROM sessions WHERE token = ?",
		token,
	).Scan(&s.ID, &s.Token, &s.UserID, &s.SessionType, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteSession(token string) error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func DeleteExpiredSessions() error {
	_, err := database.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

func IsValidSession(token string) bool {
	session, err := GetSessionByToken(token)
	if err != nil {
		return false
	}
	return session.ExpiresAt.After(time.Now())
}

// IsValidAdminSession checks if the token is a valid admin session
func IsValidAdminSession(token string) bool {
	session, err := GetSessionByToken(token)
	if err != nil {
		return false
	}
	return session.ExpiresAt.After(time.Now()) && session.SessionType == SessionTypeAdmin
}

// IsValidUserSession checks if the token is a valid user session and returns the user ID
func IsValidUserSession(token string) (int64, bool) {
	session, err := GetSessionByToken(token)
	if err != nil {
		return 0, false
	}
	if session.ExpiresAt.Before(time.Now()) {
		return 0, false
	}
	if session.SessionType != SessionTypeUser || !session.UserID.Valid {
		return 0, false
	}
	return session.UserID.Int64, true
}
