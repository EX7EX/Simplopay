package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"simplopay.com/backend/internal/transaction"
	userpkg "simplopay.com/backend/internal/user"
)

// TransactionHandler handles transaction related API requests.
type TransactionHandler struct {
	transactionService transaction.TransactionService
	userRepo           userpkg.UserRepository
}

// NewTransactionHandler creates a new TransactionHandler.
func NewTransactionHandler(transactionService transaction.TransactionService, userRepo userpkg.UserRepository) *TransactionHandler {
	return &TransactionHandler{transactionService: transactionService, userRepo: userRepo}
}

// InitiateP2PTransferRequest represents the request body for initiating a P2P transfer.
type InitiateP2PTransferRequest struct {
	ReceiverUsername string  `json:"receiver_username"`
	Amount           float64 `json:"amount"`
}

// InitiateP2PTransfer handles requests to initiate a P2P transfer.
func (h *TransactionHandler) InitiateP2PTransfer(w http.ResponseWriter, r *http.Request) {
	// 1. Parse request body
	var reqBody InitiateP2PTransferRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Prevent unknown fields
	if err := decoder.Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// 2. Basic validation
	if reqBody.ReceiverUsername == "" || reqBody.Amount <= 0 {
		http.Error(w, "Receiver username and positive amount are required", http.StatusBadRequest)
		return
	}

	// 3. Get sender user ID from context (added by auth middleware)
	senderUserID, ok := r.Context().Value(ContextKeyUserID).(string)
	if !ok || senderUserID == "" {
		// This should not happen if auth middleware is correctly applied
		http.Error(w, "Sender user ID not found in context", http.StatusInternalServerError)
		return
	}

	// 4. Find receiver user by username
	receiverUser, err := h.userRepo.FindUserByUsername(reqBody.ReceiverUsername)
	if err != nil {
		if errors.Is(err, userpkg.ErrUserNotFound) {
			http.Error(w, "Receiver user not found", http.StatusNotFound)
			return
		}
		// Handle other potential errors during user lookup
		http.Error(w, "Failed to find receiver user", http.StatusInternalServerError)
		return
	}

	// 5. Initiate the P2P transfer via the transaction service
	// The service handles self-transfer check and further validation
	orderID, _, err := h.transactionService.InitiateP2PTransfer(senderUserID, receiverUser.ID, reqBody.Amount)
	if err != nil {
		// Handle specific transaction initiation errors
		if errors.Is(err, transaction.ErrSelfTransfer) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Handle other potential errors from the transaction service
		http.Error(w, "Failed to initiate P2P transfer", http.StatusInternalServerError)
		return
	}

	// 6. Respond to the client
	// The opayResp contains the final status and any results.
	// For simplicity, just return a success message for now.
	// In a real app, you might return more details from opayResp.

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":  fmt.Sprintf("P2P Transfer initiated successfully. Order ID: %s", orderID),
		"order_id": orderID,
	})
}
