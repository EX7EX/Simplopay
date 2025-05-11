package transaction

import (
	"errors"
	"fmt"
	"log"
	"time"

	"simplopay.com/backend/internal/account"
	"simplopay.com/backend/internal/user"
	"simplopay.com/backend/pkg/opay"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	// TODO: Add a JWT library for token generation
)

var (
	// ErrUserAlreadyExists = errors.New("user already exists") // Moved to user package
	ErrInvalidCredentials = errors.New("invalid credentials")
	// TODO: Add more specific error types

	// Transaction specific errors
	ErrInvalidTransferDetails = errors.New("invalid transfer details: sender, receiver, or amount missing/invalid")
	ErrSelfTransfer           = errors.New("cannot transfer to yourself")
)

// Claims defines the structure of the JWT claims.
type Claims struct {
	// ... existing code ...
}

// TransactionService defines the interface for transaction operations.
type TransactionService interface {
	InitiateP2PTransfer(senderUserID, receiverUserID string, amount float64) (string, *opay.Response, error)
	// Add other transaction types here
}

// TransactionServiceImpl is a basic implementation of TransactionService.
type TransactionServiceImpl struct {
	opayInstance *opay.Opay
	userRepo     user.UserRepository
	accountRepo  account.AccountRepository
	// TODO: Add dependencies like UserRepository and AccountRepository
}

// NewTransactionServiceImpl creates a new TransactionServiceImpl.
func NewTransactionServiceImpl(
	opayInstance *opay.Opay,
	userRepo user.UserRepository,
	accountRepo account.AccountRepository,
) *TransactionServiceImpl {
	return &TransactionServiceImpl{
		opayInstance: opayInstance,
		userRepo:     userRepo,
		accountRepo:  accountRepo,
	}
}

// Let's go with Option 2 for clarity, defining a minimal struct for the stakeholder.
type p2pStakeholderOrder struct {
	OrderID  string
	UserID   string
	Amount   decimal.Decimal
	Currency string
	meta     *opay.Meta // Ensure this field is present and correctly named
	// TODO: Add status fields if stakeholder needs independent status tracking
}

// Implement IOrder for p2pStakeholderOrder

func (o *p2pStakeholderOrder) GetMeta() *opay.Meta {
	return o.meta
}

func (o *p2pStakeholderOrder) PreStatus() int64 {
	// Stakeholder status might not be tracked independently, return a default or zero status code
	return 0 // Or the code for an initial/unset status if applicable
}

func (o *p2pStakeholderOrder) TargetStatus() int64 {
	// Stakeholder target status might not be applicable, return a default or zero status code
	return 0 // Or the code for a target status if applicable
}

func (o *p2pStakeholderOrder) GetUid() string {
	return o.UserID // Stakeholder is the receiver
}

func (o *p2pStakeholderOrder) GetAid() string {
	return o.Currency // AID is the currency for settlement
}

func (o *p2pStakeholderOrder) GetAmount() float64 {
	amountFloat, _ := o.Amount.Float64()
	return amountFloat
}

func (o *p2pStakeholderOrder) Pend(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

func (o *p2pStakeholderOrder) Do(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

func (o *p2pStakeholderOrder) Succeed(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

func (o *p2pStakeholderOrder) Cancel(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

func (o *p2pStakeholderOrder) Fail(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

func (o *p2pStakeholderOrder) SyncDeal(tx *sqlx.Tx, addition opay.KV) error {
	// No-op for stakeholder in this simple model
	return nil
}

// InitiateP2PTransfer initiates a peer-to-peer transfer.
func (s *TransactionServiceImpl) InitiateP2PTransfer(senderUserID, receiverUserID string, amount float64) (string, *opay.Response, error) {
	// TODO: Implement P2P transfer logic using opayInstance.Do()

	// 1. Basic Validation (already partly done in handler, but reinforce here)
	if senderUserID == "" || receiverUserID == "" || amount <= 0 {
		return "", nil, ErrInvalidTransferDetails // Use specific error
	}

	// Prevent self-transfer
	if senderUserID == receiverUserID {
		return "", nil, ErrSelfTransfer // Use specific error
	}

	// Convert amount to decimal.Decimal
	decimalAmount := decimal.NewFromFloat(amount)
	if decimalAmount.IsNegative() || decimalAmount.IsZero() {
		return "", nil, ErrInvalidTransferDetails // Use specific error for amount issue
	}

	// 2. Verify Sender and Receiver Exist
	// sender, err := s.userRepo.FindUserByID(senderUserID)
	_, err := s.userRepo.FindUserByID(senderUserID) // Check existence without declaring variable
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return "", nil, fmt.Errorf("sender not found: %w", err)
		}
		return "", nil, fmt.Errorf("error finding sender: %w", err)
	}
	// receiver, err := s.userRepo.FindUserByID(receiverUserID)
	_, err = s.userRepo.FindUserByID(receiverUserID) // Check existence without declaring variable
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return "", nil, fmt.Errorf("receiver not found: %w", err)
		}
		return "", nil, fmt.Errorf("error finding receiver: %w", err)
	}

	// 3. Create the P2P Order
	// Assume NGN currency for now
	currency := "NGN" // TODO: Make this dynamic/configurable

	// Find the order meta for P2P transfer
	meta, ok := s.opayInstance.Meta("p2p_transfer") // TODO: Add Meta method to Opay struct or access directly if public
	if !ok {
		return "", nil, errors.New("p2p_transfer order type not registered") // Should not happen if registration in main is correct
	}

	orderID := uuid.New().String()
	now := time.Now()

	initiatorOrder := &P2POrder{
		OrderID:          orderID,
		SenderUserID:     senderUserID,
		ReceiverUserID:   receiverUserID,
		Amount:           decimalAmount,
		Currency:         currency,
		CurrentStatus:    int64(opay.UNSET), // Start with unset status code
		TargetStatusCode: int64(opay.PEND),  // Target pending status code
		meta:             meta,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// For P2P, the receiver is the stakeholder. The Opay framework expects a separate IOrder for the stakeholder.
	// In this simple case, the stakeholder order can be the same P2POrder instance, but with GetUid() returning receiver ID.
	// A cleaner approach might be a dedicated StakeholderP2POrder struct implementing IOrder.
	// For now, let's try passing the same initiator order, but this might need refinement based on Opay's exact stakeholder handling.
	// Let's assume Opay uses GetUid and GetAmount of the Stakeholder IOrder.
	// We need a way for the Stakeholder IOrder to return the ReceiverUserID and a positive amount.

	// Option 1: Use the same P2POrder for Stakeholder (requires P2POrder to know its role - less clean)
	// Option 2: Create a simple struct implementing IOrder for the stakeholder leg.

	// Let's go with Option 2 for clarity, defining a minimal struct for the stakeholder.
	stakeholderOrder := &p2pStakeholderOrder{
		OrderID:  orderID, // Use same order ID
		UserID:   receiverUserID,
		Amount:   decimalAmount, // Positive amount for credit
		Currency: currency,
		meta:     meta,
	}

	// 4. Create the Opay Request
	req := &opay.Request{ // Create a pointer to Request
		Initiator:   initiatorOrder,
		Stakeholder: stakeholderOrder, // Pass the stakeholder order
		// TODO: Set Deadline, Addition, Tx (Tx can be nil for Opay to manage)
	}

	// 5. Submit Request to Opay
	// opayInstance.Do() is blocking and returns the final response.
	resp := s.opayInstance.Do(req)

	// 6. Handle Opay Response
	// The P2POrder.Succeed/Fail/Cancel methods would have updated the order status in DB.
	// We can inspect resp.Err to know the outcome.

	if resp.Err != nil {
		// TODO: Log the error properly
		return "", nil, fmt.Errorf("p2p transfer failed: %w", resp.Err)
	}

	// Transfer successful
	log.Printf("P2P Transfer successful for order %s. Opay Response: %+v\n", req.Initiator.GetUid(), resp)
	return orderID, resp, nil
}
