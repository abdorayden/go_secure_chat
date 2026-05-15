package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateLogin     = errors.New("duplicate login")
	emailPattern          = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	usernamePattern       = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
)

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
}

type DBModel struct {
	db *sql.DB
}

func Open(path string, migrationPath string) (*DBModel, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	schema, err := os.ReadFile(migrationPath)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBModel{db: db}, nil
}

func (m *DBModel) Close() error {
	if m == nil || m.db == nil {
		return nil
	}
	return m.db.Close()
}

func ValidateSignup(username, email, password string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New("username must be 3-32 characters and use letters, digits, or underscore")
	}
	if !emailPattern.MatchString(email) {
		return errors.New("invalid email")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func (m *DBModel) CreateUser(ctx context.Context, username, email, password string) error {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	if err := ValidateSignup(username, email, password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = m.db.ExecContext(
		ctx,
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username,
		email,
		string(hash),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ErrUserExists
		}
		return err
	}
	return nil
}

func (m *DBModel) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := m.db.QueryRowContext(
		ctx,
		`SELECT id, username, email, password_hash FROM users WHERE email = ?`,
		strings.TrimSpace(strings.ToLower(email)),
	)
	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash); err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *DBModel) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	user, err := m.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
