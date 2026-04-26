package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joey/wcwcpp-backend/adapters/auth"
	"github.com/joey/wcwcpp-backend/adapters/handler"
	"github.com/joey/wcwcpp-backend/adapters/storage/postgres"
	"github.com/joey/wcwcpp-backend/core/service"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	// 1. Initialize Database
	db, err := postgres.NewDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Initialize Adapters
	userRepo := postgres.NewUserRepository(db)
	tokenValidator := auth.NewGoogleTokenValidator()

	// 3. Initialize Core Services
	authService := service.NewAuthService(userRepo, tokenValidator)
	contestService := service.NewContestService(postgres.NewContestRepository(db))
	usersService := service.NewUsersService()

	// 4. Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	contestHandler := handler.NewContestHandler(contestService)
	usersHandler := handler.NewUsersHandler(usersService)

	mux := http.NewServeMux()

	// 5. Register RPC Handlers to the mux
	authPath, authSvcHandler := v1connect.NewAuthServiceHandler(authHandler)
	mux.Handle(authPath, authSvcHandler)

	contestPath, contestSvcHandler := v1connect.NewContestServiceHandler(contestHandler)
	mux.Handle(contestPath, contestSvcHandler)

	usersPath, usersSvcHandler := v1connect.NewUsersServiceHandler(usersHandler)
	mux.Handle(usersPath, usersSvcHandler)

	fmt.Println("Starting server on :8080")
	// Use h2c for unencrypted HTTP/2 (required for Connect without TLS)
	err = http.ListenAndServe(
		":8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
