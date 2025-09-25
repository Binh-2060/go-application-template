package dbpkg

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func CreateDatabaseConnection() {
	var err error
	var databaseUrl = os.Getenv("DB_URL")
	DB, err = pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		panic("Failed to create connection pool:" + err.Error())
	}

	if err := DB.Ping(context.Background()); err != nil {
		panic("Failed to create connection pool:" + err.Error())
	}

	log.Println("Successfully connected to database pgx ✅!")
}
