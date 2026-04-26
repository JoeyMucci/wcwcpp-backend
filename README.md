# Testing the WCWCPP Backend in Development

This guide explains how to run the API, execute tests, and generate development authentication tokens to interact with the API endpoints locally.

## 1. Environment Setup

Make sure your database is running and the schema is applied.

```bash
# Start the PostgreSQL database via Docker Compose
make db-up

# Apply the latest schema
make db-apply
```

If you make database changes, you can regenerate the Jet query builder models using:
```bash
make db-generate
```

## 2. Running the Server

To start the API server locally (this will automatically load your `.env` file):

```bash
make run
```

## 3. Running Automated Tests

To run the complete test suite (including the integration tests that use `testcontainers-go` for database isolation), you can use the Makefile recipes:

```bash
# Run all tests
make test

# Run all tests with the race detector enabled (Recommended)
make test-race
```

## 4. Generating a Test Token for API Requests

The API requires a valid JWT for endpoints that require authentication or superadmin privileges. You can easily generate a long-lived JWT for local development testing using the `dev-token` tool.

### Creating a Token

Run the following command from the root of the project. If you don't provide an email, it defaults to `superadmin@example.com`.

```bash
# Generate a token for the default email
make token

# Generate a token for a specific email
make token EMAIL=myuser@example.com
```

### Superadmin Access

To test endpoints that are restricted to superadmins (e.g., creating a contest), ensure that the email you use to generate the token is listed in the `SUPERADMIN_EMAILS` environment variable in your `.env` file.

Example `.env`:
```env
SUPERADMIN_EMAILS=superadmin@example.com,jmucci314@gmail.com
JWT_SECRET=your_jwt_secret
```

### Using the Token

The `dev-token` script will output the exact `Authorization` header to use. Include this header in your requests (via Postman, curl, or your frontend client):

```bash
Authorization: Bearer <your-generated-jwt-string>
```
