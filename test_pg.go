package main
import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)
func main() {
	_, err := postgres.Run(context.Background(), "postgres:15-alpine")
	fmt.Println(err)
}
