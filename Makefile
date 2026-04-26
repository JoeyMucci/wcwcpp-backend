# Automatically load the .env file if it exists, and export all variables to children
-include .env
export

.PHONY: db-up db-down db-apply db-generate proto-generate test test-race run

# Start the database via Docker Compose
db-up:
	docker compose up -d

# Stop the database
db-down:
	docker compose down

# Apply the schema to the running database
db-apply:
	docker exec -i wcwcpp-db psql "${DATABASE_URL}" < db/schema.sql

# Generate Jet Go models into adapters/storage/jet
db-generate:
	@echo "Generating Jet models..."
	go run github.com/go-jet/jet/v2/cmd/jet@latest -dsn="${DATABASE_URL}" -schema=public -path=./adapters/storage/jet
	@echo "Done!"

# Generate Protobuf code
proto-generate:
	@echo "Generating Protobuf code..."
	go run github.com/bufbuild/buf/cmd/buf@latest generate
	@echo "Done!"

# Run all Go tests
test:
	go test -v ./...

# Run all Go tests with race detector
test-race:
	go test -race -v ./...

# Run the API server locally
run:
	go run cmd/server/main.go

# Generate a dev token with env vars loaded. Usage: make token [EMAIL=user@example.com]
token:
	go run cmd/dev-token/main.go $(EMAIL)
