package database

import (
	"database/sql"
	"errors"
	"fmt"

	userpkg "simplopay.com/backend/internal/user"

	"github.com/lib/pq"
)

// UserRepositoryImpl is a database implementation of UserRepository.
type UserRepositoryImpl struct {
	db *sql.DB
}

// NewUserRepositoryImpl creates a new UserRepositoryImpl.
func NewUserRepositoryImpl(db *sql.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

// CreateUser creates a new user in the database.
func (r *UserRepositoryImpl) CreateUser(u *userpkg.User) error {
	query := `INSERT INTO users (id, username, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(query, u.ID, u.Username, u.Password, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return userpkg.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindUserByUsername finds a user by username in the database.
func (r *UserRepositoryImpl) FindUserByUsername(username string) (*userpkg.User, error) {
	query := `SELECT id, username, password, created_at, updated_at FROM users WHERE username = $1`

	row := r.db.QueryRow(query, username)

	var u userpkg.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, userpkg.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return &u, nil
}

// FindUserByID finds a user by ID in the database.
func (r *UserRepositoryImpl) FindUserByID(id string) (*userpkg.User, error) {
	query := `SELECT id, username, password, created_at, updated_at FROM users WHERE id = $1`

	row := r.db.QueryRow(query, id)

	var u userpkg.User
	err := row.Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, userpkg.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return &u, nil
}
