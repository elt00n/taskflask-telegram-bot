package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultUserTimezone = "UTC"

var (
	ErrUserIDRequired           = errors.New("user ID is required")
	ErrUserFirstNameRequired    = errors.New("user first name is required")
	ErrInvalidUserTimezone      = errors.New("invalid user timezone")
	ErrUserCreationTimeRequired = errors.New("user creation time is required")
)

// User представляет известного боту пользователя Telegram.
type User struct {
	ID        UserID
	Username  string
	FirstName string
	Timezone  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUserParams группирует входные данные нового пользователя.
type NewUserParams struct {
	ID        UserID
	Username  string
	FirstName string
	Timezone  string
}

// NewUser проверяет профиль и применяет часовой пояс UTC по умолчанию.
func NewUser(params NewUserParams, now time.Time) (User, error) {
	if !params.ID.IsValid() {
		return User{}, ErrUserIDRequired
	}

	firstName := strings.TrimSpace(params.FirstName)
	if firstName == "" {
		return User{}, ErrUserFirstNameRequired
	}
	if now.IsZero() {
		return User{}, ErrUserCreationTimeRequired
	}

	timezone := strings.TrimSpace(params.Timezone)
	if timezone == "" {
		timezone = DefaultUserTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return User{}, fmt.Errorf("%w: %s", ErrInvalidUserTimezone, timezone)
	}

	now = now.UTC()

	return User{
		ID:        params.ID,
		Username:  strings.TrimPrefix(strings.TrimSpace(params.Username), "@"),
		FirstName: firstName,
		Timezone:  timezone,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
