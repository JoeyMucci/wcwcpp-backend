package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/joey/wcwcpp-backend/adapters/auth"
	"github.com/joey/wcwcpp-backend/adapters/handler"
	"github.com/joey/wcwcpp-backend/adapters/storage/postgres"
	"github.com/joey/wcwcpp-backend/core/service"
	"github.com/joey/wcwcpp-backend/pkg/api/v1/v1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type customJSONCodec struct {
	marshalOpts   protojson.MarshalOptions
	unmarshalOpts protojson.UnmarshalOptions
}

func (c *customJSONCodec) Name() string {
	return "json"
}

func (c *customJSONCodec) Marshal(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("message is not a proto.Message")
	}
	return c.marshalOpts.Marshal(protoMessage)
}

func (c *customJSONCodec) Unmarshal(data []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("message is not a proto.Message")
	}
	return c.unmarshalOpts.Unmarshal(data, protoMessage)
}

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
	contestRepo := postgres.NewContestRepository(db)
	contestService := service.NewContestService(contestRepo)
	usersService := service.NewUsersService(userRepo)
	matchService := service.NewMatchService(contestRepo)

	// 4. Initialize Handlers
	authHandler := handler.NewAuthHandler(authService)
	contestHandler := handler.NewContestHandler(contestService)
	usersHandler := handler.NewUsersHandler(usersService)
	matchHandler := handler.NewMatchHandler(matchService)

	leaderboardRepo := postgres.NewLeaderboardRepository(db)
	leaderboardService := service.NewLeaderboardService(leaderboardRepo)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)

	picksRepo := postgres.NewPicksRepository(db)
	picksService := service.NewPicksService(picksRepo)
	picksHandler := handler.NewPicksHandler(picksService)

	// Custom JSON codec to emit unpopulated fields (e.g. including points: 0, wins: 0 in response)
	jsonCodec := &customJSONCodec{
		marshalOpts: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		unmarshalOpts: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}

	mux := http.NewServeMux()

	// 5. Register RPC Handlers to the mux with custom JSON codec
	authPath, authSvcHandler := v1connect.NewAuthServiceHandler(authHandler, connect.WithCodec(jsonCodec))
	mux.Handle(authPath, authSvcHandler)

	contestPath, contestSvcHandler := v1connect.NewContestServiceHandler(contestHandler, connect.WithCodec(jsonCodec))
	mux.Handle(contestPath, contestSvcHandler)

	usersPath, usersSvcHandler := v1connect.NewUsersServiceHandler(usersHandler, connect.WithCodec(jsonCodec))
	mux.Handle(usersPath, usersSvcHandler)

	leaderboardPath, leaderboardSvcHandler := v1connect.NewLeaderboardServiceHandler(leaderboardHandler, connect.WithCodec(jsonCodec))
	mux.Handle(leaderboardPath, leaderboardSvcHandler)

	picksPath, picksSvcHandler := v1connect.NewPicksServiceHandler(picksHandler, connect.WithCodec(jsonCodec))
	mux.Handle(picksPath, picksSvcHandler)

	matchPath, matchSvcHandler := v1connect.NewMatchServiceHandler(matchHandler, connect.WithCodec(jsonCodec))
	mux.Handle(matchPath, matchSvcHandler)

	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsStr == "" {
		log.Fatal("ALLOWED_ORIGINS environment variable is required but not set")
	}

	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	for i, origin := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(origin)
	}

	// Simple and robust CORS middleware to support preflight OPTIONS requests
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqOrigin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
			for _, allowed := range allowedOrigins {
				cleanAllowed := strings.TrimRight(strings.TrimSpace(allowed), "/")
				if strings.EqualFold(reqOrigin, cleanAllowed) {
					w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					break
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			
			// Dynamically allow requested headers to be resilient to browser/library-specific differences
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Connect-Protocol-Version")
			}
			
			w.Header().Set("Access-Control-Expose-Headers", "Connect-Error-Info, Connect-Protocol-Version, Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is required but not set")
	}

	// Create a production-ready HTTP server with h2c and timeouts
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(corsMiddleware(mux), &http2.Server{}),
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("Starting server on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for termination signal
	<-stop
	fmt.Println("Shutting down server gracefully...")

	// Create context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close database connection pool
	if err := db.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}

	fmt.Println("Server stopped successfully")
}
