package account

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

var (
	ErrAccountNotFound = errors.New("account not found")
)

// Account represents a user's financial account/wallet.
type Account struct {
	ID        string          `db:"id"`
	UserID    string          `db:"user_id"`
	Currency  string          `db:"currency"`
	Balance   decimal.Decimal `db:"balance"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

// AccountRepository defines the interface for account data operations.
type AccountRepository interface {
	CreateAccount(account *Account) error
	FindAccountByID(id string) (*Account, error)
	FindAccountsByUserID(userID string) ([]*Account, error)
	// UpdateAccountBalance updates an account's balance within a transaction.
	UpdateAccountBalance(tx *sqlx.Tx, id string, balanceChange decimal.Decimal) error
	// FindAccountByUserIDAndCurrency finds a user's account for a specific currency.
	FindAccountByUserIDAndCurrency(userID, currency string) (*Account, error)
	// Add other necessary methods.
}
