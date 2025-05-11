package auth

import (
	"errors"
	"fmt"
	"time"

	"simplopay.com/backend/internal/account"
	"simplopay.com/backend/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	// TODO: Add a JWT library for token generation
)

var (
	// ErrUserAlreadyExists = errors.New("user already exists") // Moved to user package
	ErrInvalidCredentials = errors.New("invalid credentials")
	// TODO: Add more specific error types
)

// Claims defines the structure of the JWT claims.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthService defines the interface for authentication operations.
type AuthService interface {
	RegisterUser(username, password string) (*user.User, error)
	Login(username, password string) (string, error) // Returns authentication token
	// Add other necessary methods like ChangePassword, etc.
}

// AuthServiceImpl is a basic implementation of AuthService.
type AuthServiceImpl struct {
	userRepo      user.UserRepository
	accountRepo   account.AccountRepository
	jwtSecretKey  []byte
	tokenDuration time.Duration
}

// NewAuthServiceImpl creates a new AuthServiceImpl.
func NewAuthServiceImpl(
	userRepo user.UserRepository,
	accountRepo account.AccountRepository,
	jwtSecretKey []byte,
	tokenDuration time.Duration,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		userRepo:      userRepo,
		accountRepo:   accountRepo,
		jwtSecretKey:  jwtSecretKey,
		tokenDuration: tokenDuration,
	}
}

// RegisterUser a new user.
func (s *AuthServiceImpl) RegisterUser(username, password string) (*user.User, error) {
	// Check if user already exists using userRepo.FindUserByUsername
	existingUser, err := s.userRepo.FindUserByUsername(username)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		// Handle database error that is not ErrUserNotFound
		return nil, fmt.Errorf("error checking for existing user: %w", err)
	}
	if existingUser != nil {
		return nil, user.ErrUserAlreadyExists
	}

	// Hash the password securely using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser := &user.User{
		ID:        uuid.New().String(), // Generate a unique ID
		Username:  username,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.userRepo.CreateUser(newUser)
	if err != nil {
		// TODO: Handle specific database errors if necessary, although userRepo should handle unique_violation
		return nil, fmt.Errorf("failed to create user in repository: %w", err)
	}

	// Create a default account for the new user
	defaultAccount := &account.Account{
		ID:        uuid.New().String(), // Generate a unique ID for the account
		UserID:    newUser.ID,
		Currency:  "NGN",        // Default currency for now, will be configurable
		Balance:   decimal.Zero, // Initial balance is zero
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.accountRepo.CreateAccount(defaultAccount)
	if err != nil {
		// TODO: Handle error creating account, potentially compensate by deleting the created user
		// This requires adding a DeleteUser method to the UserRepository interface and implementation
		return nil, fmt.Errorf("failed to create default account: %w", err)
	}

	return newUser, nil
}

// Login a user and return an authentication token.
func (s *AuthServiceImpl) Login(username, password string) (string, error) {
	// Find user by username
	foundUser, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("error finding user during login: %w", err)
	}

	// Compare provided password with hashed password
	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("error comparing passwords: %w", err)
	}

	// Generate JWT token
	expirationTime := time.Now().Add(s.tokenDuration)
	claims := &Claims{
		UserID: foundUser.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "simplopay", // TODO: Make issuer configurable
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("error generating token: %w", err)
	}

	return tokenString, nil
}
