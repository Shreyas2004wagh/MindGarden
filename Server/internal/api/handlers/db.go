package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	db     *sql.DB
	dbOnce sync.Once
)

// InitDB initializes the database connection
func InitDB() error {
	var err error
	dbOnce.Do(func() {
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			// Construct from individual parts if DATABASE_URL not set
			host := os.Getenv("DB_HOST")
			port := os.Getenv("DB_PORT")
			user := os.Getenv("DB_USER")
			password := os.Getenv("DB_PASSWORD")
			dbname := os.Getenv("DB_NAME")

			if host == "" || user == "" || password == "" || dbname == "" {
				err = fmt.Errorf("database configuration missing. Set DATABASE_URL or DB_* environment variables")
				return
			}

			if port == "" {
				port = "5432"
			}

			dbURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
				host, port, user, password, dbname)
		}

		db, err = sql.Open("pgx", dbURL)
		if err != nil {
			err = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// Set a timeout for the ping
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test connection
		if err = db.PingContext(ctx); err != nil {
			err = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		// Set connection pool settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)

		log.Println("Database connection established")
	})

	return err
}

// getDB returns the database instance
func getDB() *sql.DB {
	return db
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// insertJournal inserts a new journal entry into the database
func insertJournal(ctx context.Context, journalID, userID string, title *string, content string, createdAt time.Time) (*JournalResponse, error) {
	database := getDB()
	if database == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	query := `
		INSERT INTO journals (id, user_id, title, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, user_id, title, content, created_at, updated_at
	`

	var journal JournalResponse
	err := database.QueryRowContext(ctx, query, journalID, userID, title, content, createdAt).Scan(
		&journal.ID,
		&journal.UserID,
		&journal.Title,
		&journal.Content,
		&journal.CreatedAt,
		&journal.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert journal: %w", err)
	}

	return &journal, nil
}
