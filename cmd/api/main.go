package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"simplopay.com/backend/api/handler"
	"simplopay.com/backend/internal/account"
	"simplopay.com/backend/internal/auth"
	"simplopay.com/backend/internal/database"
	"simplopay.com/backend/internal/transaction"
	"simplopay.com/backend/pkg/opay"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ContextKey represents the type for context keys.
type ContextKey string

const (
	ContextKeyUserID ContextKey = "userID"
)

func main() {
	// TODO: Load configuration (e.g., database URL, JWT secret)
	dbURL := "postgres://user:password@host:port/dbname?sslmode=disable" // Placeholder
	jwtSecret := []byte("your-very-secure-jwt-secret")                   // Placeholder
	tokenDuration := time.Hour * 24                                      // Placeholder
	// TODO: Configure Opay queue capacity and decimal places
	opayQueueCapacity := 100
	opayDecimalPlaces := 2

	// Database connection
	// Use sqlx.Connect for easier integration with sqlx types in Opay
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// TODO: Ping database to verify connection

	// Repositories
	userRepo := database.NewUserRepositoryImpl(db.DB)
	accountRepo := database.NewAccountRepositoryImpl(db)

	// Services
	authService := auth.NewAuthServiceImpl(
		userRepo,
		accountRepo,
		jwtSecret,
		tokenDuration,
	)
	// Pass sqlx.DB to Opay
	opayInstance := opay.NewOpay(db, opayQueueCapacity, opayDecimalPlaces)
	// TODO: Pass necessary repositories to TransactionServiceImpl
	transactionService := transaction.NewTransactionServiceImpl(opayInstance, userRepo, accountRepo)

	// Register Opay Handlers (Order Types)
	// Define statuses for P2P transfer
	p2pStatuses := []opay.Status{
		{Code: 1, Note: "P2P Transfer Pending", Step: opay.PEND},
		{Code: 2, Note: "P2P Transfer In Progress", Step: opay.DO},
		{Code: 3, Note: "P2P Transfer Succeeded", Step: opay.SUCCEED},
		{Code: 4, Note: "P2P Transfer Failed", Step: opay.FAIL},
		{Code: 5, Note: "P2P Transfer Cancelled", Step: opay.CANCEL},
	}

	// Create and register P2P handler
	p2pHandler := transaction.NewP2PHandler(accountRepo, userRepo)

	_, err = opayInstance.RegMeta("p2p_transfer", p2pHandler, p2pStatuses)
	if err != nil {
		log.Fatalf("Failed to register P2P transfer meta: %v", err)
	}

	// Register Opay SettleFuncs (Account Operations)
	internalSettleService := account.NewInternalSettleService(accountRepo)
	nibssSettleService := account.NewNIBSSSettleServiceImpl()

	err = opay.RegSettleFunc("NGN", internalSettleService.UpdateBalance)
	if err != nil {
		log.Fatalf("Failed to register NGN settle function: %v", err)
	}

	// TODO: Define a currency code for NIBSS external accounts, e.g., "NIBSS_NGN"
	err = opay.RegSettleFunc("NIBSS_NGN", nibssSettleService.UpdateBalance)
	if err != nil {
		log.Fatalf("Failed to register NIBSS settle function: %v", err)
	}

	// Start Opay service
	go opayInstance.Serve()

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	transactionHandler := handler.NewTransactionHandler(transactionService, userRepo)

	// Router
	r := mux.NewRouter()

	// Add middleware
	r.Use(loggingMiddleware)

	// Define public routes (no authentication required)
	publicRouter := r.PathPrefix("/auth").Subrouter()
	publicRouter.HandleFunc("/register", authHandler.Register).Methods("POST")
	publicRouter.HandleFunc("/login", authHandler.Login).Methods("POST")

	// Define protected routes (authentication required)
	protectedRouter := r.PathPrefix("/api").Subrouter()
	protectedRouter.Use(authMiddleware(jwtSecret))

	// Add protected routes here
	protectedRouter.HandleFunc("/transactions/p2p", transactionHandler.InitiateP2PTransfer).Methods("POST")

	// Start server
	port := ":8080"
	fmt.Printf("Server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}

// loggingMiddleware logs incoming HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

// authMiddleware authenticates requests using JWT tokens.
func authMiddleware(jwtSecretKey []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check if the header is in the format "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Authorization header format must be Bearer <token>", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Parse and validate the token
			claims := &auth.Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return jwtSecretKey, nil
			})

			if err != nil {
				if errors.Is(err, jwt.ErrSignatureInvalid) {
					http.Error(w, "Invalid token signature", http.StatusUnauthorized)
					return
				}
				// Handle expired tokens and other parsing errors
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add user ID to context
			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
