package transaction

import (
	"errors"
	"fmt"

	"simplopay.com/backend/pkg/opay"

	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	// TODO: Import necessary internal packages like user and account
)

var (
	ErrAccountNotFound = errors.New("account not found") // This should ideally be in internal/account
)

// P2POrder represents a peer-to-peer transfer order.
type P2POrder struct {
	OrderID          string          // Unique ID for this order
	SenderUserID     string          // User ID of the sender
	ReceiverUserID   string          // User ID of the receiver
	Amount           decimal.Decimal // Amount of the transaction
	Currency         string          // Currency of the transaction
	CurrentStatus    int64           // Current status code
	TargetStatusCode int64           // Target status code for the requested operation
	meta             *opay.Meta      // Reference to the order meta
	CreatedAt        time.Time       // Timestamp when the order was created
	UpdatedAt        time.Time       // Timestamp when the order was last updated

	// Temporarily hold the transaction and account repository for methods
	// TODO: Ideally, these methods should interact with a dedicated OrderRepository
	tx *sqlx.Tx
}

// Ensure P2POrder implements opay.IOrder
var _ opay.IOrder = (*P2POrder)(nil)

// GetMeta returns the metadata for this order type.
func (o *P2POrder) GetMeta() *opay.Meta {
	return o.meta
}

// PreStatus returns the status code before the requested operation.
func (o *P2POrder) PreStatus() int64 {
	return o.CurrentStatus
}

// TargetStatus returns the status code for the requested operation (implements opay.IOrder).
func (o *P2POrder) TargetStatus() int64 {
	return o.TargetStatusCode
}

// GetUid returns the user ID associated with the initiator of this order (the sender).
func (o *P2POrder) GetUid() string {
	return o.SenderUserID
}

// GetAid returns the asset ID (currency) for this order.
func (o *P2POrder) GetAid() string {
	return o.Currency
}

// GetAmount returns the transaction amount.
func (o *P2POrder) GetAmount() float64 {
	// Return the absolute value as float64 for the Opay framework.
	// The SettleFunc needs to handle debit/credit based on whether it's for initiator or stakeholder.
	amountFloat, _ := o.Amount.Abs().Float64()
	return amountFloat
}

// updateStatusInDB is a helper to update the order's status in the database.
// TODO: Replace with interaction with a dedicated OrderRepository.
func (o *P2POrder) updateStatusInDB(tx *sqlx.Tx, status int64) error {
	query := `UPDATE orders SET current_status = $1, updated_at = $2 WHERE order_id = $3`
	_, err := tx.Exec(query, status, time.Now(), o.OrderID)
	if err != nil {
		return fmt.Errorf("failed to update order status to %d for order %s: %w", status, o.OrderID, err)
	}
	o.CurrentStatus = status
	return nil
}

// Pend marks the order as pending and updates the status in the database.
func (o *P2POrder) Pend(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// Update status in DB to pending
	// The specific pending status code should be defined in our P2P statuses
	pendingStatusCode := int64(opay.PEND) // Using opay.PEND value directly for simplicity, match with registered statuses
	return o.updateStatusInDB(tx, pendingStatusCode)
}

// Do marks the order as being processed and updates the status in the database.
func (o *P2POrder) Do(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// Update status in DB to doing
	doingStatusCode := int64(opay.DO) // Using opay.DO value directly for simplicity
	return o.updateStatusInDB(tx, doingStatusCode)
}

// Succeed processes the account and marks the order as successful.
func (o *P2POrder) Succeed(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// In the Opay framework, the Context handles calling UpdateBalance for initiator and stakeholder
	// after the Do or SyncDeal methods. We just need to update the order status here.

	// Update order status in DB to successful
	successStatusCode := int64(opay.SUCCEED) // Using opay.SUCCEED value directly for simplicity
	return o.updateStatusInDB(tx, successStatusCode)
}

// Cancel marks the order as Canceled and updates the status in the database.
func (o *P2POrder) Cancel(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// Update status in DB to canceled
	// If balances were already updated (e.g., in a previous step before failure/cancel),
	// you might need to trigger a rollback via the handler or a separate Opay request.
	cancelStatusCode := int64(opay.CANCEL) // Using opay.CANCEL value directly for simplicity
	return o.updateStatusInDB(tx, cancelStatusCode)
}

// Fail marks the order as failed and updates the status in the database.
func (o *P2POrder) Fail(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// Update status in DB to failed
	// Similar to Cancel, consider rolling back balances if necessary.
	failStatusCode := int64(opay.FAIL) // Using opay.FAIL value directly for simplicity
	return o.updateStatusInDB(tx, failStatusCode)
}

// SyncDeal processes the order synchronously, updates balances, and marks it as successful.
func (o *P2POrder) SyncDeal(tx *sqlx.Tx, kv opay.KV) error {
	// o.tx = tx // Remove temporary transaction storage
	// This method is for synchronous processing. It should combine the logic of Do and Succeed.
	// Update status to doing
	doingStatusCode := int64(opay.DO)
	err := o.updateStatusInDB(tx, doingStatusCode)
	if err != nil {
		return err
	}

	// In synchronous processing, the handler/context should ideally manage the balance update call after this step.

	// Update status to successful
	successStatusCode := int64(opay.SUCCEED)
	return o.updateStatusInDB(tx, successStatusCode)
}
