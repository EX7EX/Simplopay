package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"simplopay.com/backend/internal/account"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// AccountRepositoryImpl is a database implementation of AccountRepository.
type AccountRepositoryImpl struct {
	db *sqlx.DB // Use sqlx.DB for transactional operations
}

// NewAccountRepositoryImpl creates a new AccountRepositoryImpl.
func NewAccountRepositoryImpl(db *sqlx.DB) *AccountRepositoryImpl {
	return &AccountRepositoryImpl{db: db}
}

// CreateAccount creates a new account in the database.
func (r *AccountRepositoryImpl) CreateAccount(account *account.Account) error {
	query := `INSERT INTO accounts (id, user_id, currency, balance, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	// Use r.db.Exec for non-transactional insert or ensure transaction is handled externally
	_, err := r.db.Exec(query, account.ID, account.UserID, account.Currency, account.Balance, account.CreatedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}
	return nil
}

// FindAccountByID finds an account by ID in the database.
func (r *AccountRepositoryImpl) FindAccountByID(id string) (*account.Account, error) {
	query := `SELECT id, user_id, currency, balance, created_at, updated_at FROM accounts WHERE id = $1`

	var acc account.Account
	err := r.db.Get(&acc, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, account.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by ID: %w", err)
	}

	return &acc, nil
}

// FindAccountsByUserID finds accounts by user ID in the database.
func (r *AccountRepositoryImpl) FindAccountsByUserID(userID string) ([]*account.Account, error) {
	query := `SELECT id, user_id, currency, balance, created_at, updated_at FROM accounts WHERE user_id = $1`

	accounts := []*account.Account{}
	err := r.db.Select(&accounts, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find accounts by user ID: %w", err)
	}

	return accounts, nil
}

// UpdateAccountBalance updates an account's balance within a transaction.
func (r *AccountRepositoryImpl) UpdateAccountBalance(tx *sqlx.Tx, id string, balanceChange decimal.Decimal) error {
	query := `UPDATE accounts SET balance = balance + $1, updated_at = $2 WHERE id = $3`
	result, err := tx.Exec(query, balanceChange, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update account balance within transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected after updating account balance within transaction: %w", err)
	}

	if rowsAffected == 0 {
		// This indicates the account with the given ID was not found within the transaction
		return fmt.Errorf("account with ID %s not found for balance update", id)
	}

	return nil
}

// FindAccountByUserIDAndCurrency finds a user's account for a specific currency.
func (r *AccountRepositoryImpl) FindAccountByUserIDAndCurrency(userID, currency string) (*account.Account, error) {
	query := `SELECT id, user_id, currency, balance, created_at, updated_at FROM accounts WHERE user_id = $1 AND currency = $2`

	var acc account.Account
	err := r.db.Get(&acc, query, userID, currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, account.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to find account by user ID and currency: %w", err)
	}

	return &acc, nil
}
