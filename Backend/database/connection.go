package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	Type             string `json:"type"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	User             string `json:"user"`
	Password         string `json:"password"`
	DBName           string `json:"dbname"`
	SSLMode          string `json:"sslmode"`
	ConnectionString string `json:"connection_string"`
}

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection
func NewDB(config Config) (*DB, error) {
	// Connect to PostgreSQL
	conn, err := sql.Open("postgres", config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	// Test the connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	db := &DB{
		conn: conn,
	}

	// Create tables if they don't exist
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	// Run migrations to ensure database schema is up to date
	if err := db.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	if err := db.ensureDevTestUser(); err != nil {
		log.Printf("Warning: Failed to seed live test user: %v", err)
	}

	log.Println("Database connection established successfully")
	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// HealthCheck checks if the database connection is healthy
func (db *DB) HealthCheck() error {
	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.conn.Ping()
}

func (db *DB) Stats() sql.DBStats {
	return db.conn.Stats()
}
