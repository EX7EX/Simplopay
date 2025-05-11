package account

import (
	"errors"
	"fmt"

	"simplopay.com/backend/pkg/opay"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// InternalSettleService implements opay.SettleFunc for internal accounts.
type InternalSettleService struct {
	accountRepo AccountRepository
	// TODO: Maybe add a way to get the default currency or pass it in
}

// NewInternalSettleService creates a new InternalSettleService.
func NewInternalSettleService(accountRepo AccountRepository) *InternalSettleService {
	return &InternalSettleService{accountRepo: accountRepo}
}

// UpdateBalance is the SettleFunc implementation for internal accounts.
// It updates the balance of the user's account for the default currency within the provided transaction.
func (s *InternalSettleService) UpdateBalance(uid string, amount float64, tx *sqlx.Tx) error {
	// Assuming uid is the UserID and we are dealing with the default currency (NGN)
	userID := uid
	currency := "NGN" // TODO: Make this configurable

	// Convert float64 amount to decimal.Decimal
	decimalAmount := decimal.NewFromFloat(amount)

	// Find the user's account by UserID and Currency
	acc, err := s.accountRepo.FindAccountByUserIDAndCurrency(userID, currency)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			// This could indicate a problem if a user is expected to have a default account
			return fmt.Errorf("internal account not found for user %s and currency %s: %w", userID, currency, err)
		}
		return fmt.Errorf("failed to find internal account for user %s and currency %s: %w", userID, currency, err)
	}

	// Update the account balance using the provided transaction
	err = s.accountRepo.UpdateAccountBalance(tx, acc.ID, decimalAmount)
	if err != nil {
		return fmt.Errorf("failed to update balance for account %s: %w", acc.ID, err)
	}

	return nil
}

// Ensure UpdateBalance has the opay.SettleFunc signature.
var _ opay.SettleFunc = (*InternalSettleService)(nil).UpdateBalance

// TODO: Implement other necessary account-related logic here or in a separate service
