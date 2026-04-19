# Automatically load the .env file if it exists, and export all variables to children
-include .env
export

.PHONY: db-up db-down db-apply db-generate proto-generate

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
