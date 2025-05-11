package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"simplopay.com/backend/internal/auth"
	userpkg "simplopay.com/backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

// ContextKey represents the type for context keys.
type ContextKey string

const (
	ContextKeyUserID ContextKey = "userID"
)

// AuthHandler handles authentication related HTTP requests.
type AuthHandler struct {
	authService auth.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService auth.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResponse represents the response body for user registration.
type RegisterResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response body for user login.
type LoginResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

// Register handles user registration.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	_, err := h.authService.RegisterUser(req.Username, req.Password)
	if err != nil {
		// Check for specific errors and return appropriate status codes
		if errors.Is(err, userpkg.ErrUserAlreadyExists) {
			http.Error(w, "User with this username already exists", http.StatusConflict)
			return
		}
		// Handle other potential errors during registration
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// Login handles user login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		// Check for specific errors
		if errors.Is(err, userpkg.ErrUserNotFound) || errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		// Handle other potential login errors
		http.Error(w, "Failed to login", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
