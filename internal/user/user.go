package user

import (
	"errors"
	"time"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

// User represents a user in the system.
type User struct {
	ID        string    `db:"id"`
	Username  string    `db:"username"`
	Password  string    `db:"password"` // Storing hashed password
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// UserRepository defines the interface for interacting with user data.
type UserRepository interface {
	CreateUser(user *User) error
	FindUserByUsername(username string) (*User, error)
	FindUserByID(id string) (*User, error)
	// Add other user data access methods here
}
