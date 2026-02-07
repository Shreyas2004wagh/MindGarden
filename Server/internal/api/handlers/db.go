package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
	dbOnce sync.Once
)

// InitDB initializes the database connection using pgxpool
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

			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				user, password, host, port, dbname)
		}

		config, parseErr := pgxpool.ParseConfig(dbURL)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse database config: %w", parseErr)
			return
		}

		// Configure pool settings as requested
		config.MaxConns = 25
		config.MinConns = 5
		config.MaxConnLifetime = time.Hour
		config.MaxConnIdleTime = 30 * time.Minute

		// Create the pool
		dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			err = fmt.Errorf("failed to create database pool: %w", err)
			return
		}

		// Test connection
		if err = dbPool.Ping(context.Background()); err != nil {
			err = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		log.Println("Database connection established with pgxpool")
	})

	return err
}

// GetDB returns the database pool instance
// Note: This matches the casing of the previous getDB but assumes public access might be needed, 
// though for this package private getDB was used. 
// Refactoring to public GetDB is safer for cross-package use if needed, 
// but keeping consistent with existing usage pattern in this file.
func GetDB() *pgxpool.Pool {
	return dbPool
}

// CloseDB closes the database connection
func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
	}
}

// insertJournal inserts a new journal entry into the database
func insertJournal(ctx context.Context, journalID, userID string, title *string, content string, createdAt time.Time) (*JournalResponse, error) {
	query := `
		INSERT INTO journals (id, user_id, title, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, user_id, title, content, created_at, updated_at
	`

	var journal JournalResponse
	// pgx uses QueryRow, but scanning is slightly different (no .Scan on the row itself directly in the same way with stdlib compatibility wrappers, 
	// but pgxnative supports .Scan). 
	// However, since we are using pgxpool, we use QueryRow.
	
	err := GetDB().QueryRow(ctx, query, journalID, userID, title, content, createdAt).Scan(
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
