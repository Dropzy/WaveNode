package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrLastAdministrator = errors.New("the server must keep at least one administrator")

// User operations

// GetAllUsers retrieves all users from the database
func (db *DB) GetAllUsers() ([]User, error) {
	query := `SELECT id, username, email, role, password, created_at, updated_at FROM users ORDER BY username`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Password, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %v", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %v", err)
	}

	return users, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id string) (*User, error) {
	query := `SELECT id, username, email, role, password, created_at, updated_at FROM users WHERE id = $1`

	var user User
	err := db.conn.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, email, role, password, created_at, updated_at FROM users WHERE username = $1`

	var user User
	err := db.conn.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, username, email, role, password, created_at, updated_at FROM users WHERE email = $1`

	var user User
	err := db.conn.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Role, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	return &user, nil
}

// CreateUser creates a new user with a hashed password
func (db *DB) CreateUser(user *User, password string) error {
	// Generate ID if not provided
	if user.ID == "" {
		user.ID = fmt.Sprintf("user_%d", time.Now().UnixNano())
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, username, email, role, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = db.conn.Exec(query, user.ID, user.Username, user.Email, user.Role, user.Password, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

// UpdateUser updates an existing user
func (db *DB) UpdateUser(user *User) error {
	query := `
		UPDATE users SET 
			username = $2, 
			email = $3, 
			role = $4, 
			password = $5, 
			updated_at = $6
		WHERE id = $1
	`

	user.UpdatedAt = time.Now()

	result, err := db.conn.Exec(query, user.ID, user.Username, user.Email, user.Role, user.Password, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserRole updates only the role of a user
func (db *DB) UpdateUserRole(userID, role string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin user role update: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock(8751423902)"); err != nil {
		return fmt.Errorf("failed to lock administrator changes: %v", err)
	}

	var currentRole string
	if err := tx.QueryRow("SELECT role FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&currentRole); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to load user role: %v", err)
	}
	if currentRole == "admin" && role != "admin" {
		var adminCount int
		if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount); err != nil {
			return fmt.Errorf("failed to count administrators: %v", err)
		}
		if adminCount <= 1 {
			return ErrLastAdministrator
		}
	}

	result, err := tx.Exec(`UPDATE users SET role = $2, updated_at = $3 WHERE id = $1`, userID, role, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update user role: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return tx.Commit()
}

// DeleteUser removes a user from the database
func (db *DB) DeleteUser(id string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin user deletion: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock(8751423902)"); err != nil {
		return fmt.Errorf("failed to lock administrator changes: %v", err)
	}

	var role string
	if err := tx.QueryRow("SELECT role FROM users WHERE id = $1 FOR UPDATE", id).Scan(&role); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to load user: %v", err)
	}
	if role == "admin" {
		var adminCount int
		if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount); err != nil {
			return fmt.Errorf("failed to count administrators: %v", err)
		}
		if adminCount <= 1 {
			return ErrLastAdministrator
		}
	}

	result, err := tx.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return tx.Commit()
}

// ValidatePassword validates a user's password by username and returns the user if valid
func (db *DB) ValidatePassword(username, password string) (*User, error) {
	// Get user by username
	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Validate password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	return user, nil
}

// ValidatePasswordForUser validates a user's password
func (db *DB) ValidatePasswordForUser(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (db *DB) UpdateUserPassword(userID, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to secure password: %v", err)
	}
	result, err := db.conn.Exec(`
		UPDATE users SET password = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, userID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
