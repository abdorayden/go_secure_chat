package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

type MessageHistoryRecord struct {
	MessageID         string
	SenderUsername    string
	RecipientUsername string
	Ciphertext        string
	EncryptedKey      string
	CreatedAtUnix     int64
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
	if err := migrateLegacyUsersSchema(db); err != nil {
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

func (m *DBModel) SaveMessageHistory(ctx context.Context, messageID, senderUsername, ciphertext string, encryptedKeys map[string]string, timestamp int64) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO message_history
			(message_id, sender_username, recipient_username, ciphertext, encrypted_key, created_at)
		VALUES (?, ?, ?, ?, ?, datetime(?, 'unixepoch'))
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for recipientUsername, encryptedKey := range encryptedKeys {
		if _, err := stmt.ExecContext(ctx, messageID, senderUsername, recipientUsername, ciphertext, encryptedKey, timestamp); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *DBModel) LoadMessageHistoryForUser(ctx context.Context, username string, limit int) ([]MessageHistoryRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := m.db.QueryContext(ctx, `
		SELECT message_id, sender_username, recipient_username, ciphertext, encrypted_key, strftime('%s', created_at)
		FROM message_history
		WHERE recipient_username = ?
		ORDER BY created_at ASC, id ASC
		LIMIT ?
	`, username, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]MessageHistoryRecord, 0, limit)
	for rows.Next() {
		var (
			record        MessageHistoryRecord
			createdAtUnix string
		)
		if err := rows.Scan(
			&record.MessageID,
			&record.SenderUsername,
			&record.RecipientUsername,
			&record.Ciphertext,
			&record.EncryptedKey,
			&createdAtUnix,
		); err != nil {
			return nil, err
		}
		if createdAtUnix != "" {
			fmt.Sscanf(createdAtUnix, "%d", &record.CreatedAtUnix)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func migrateLegacyUsersSchema(db *sql.DB) error {
	columns, err := userTableColumns(db)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	hasEmail := columns["email"]
	hasPasswordHash := columns["password_hash"]
	hasMail := columns["mail"]
	hasPassword := columns["password"]

	if hasEmail && hasPasswordHash {
		return nil
	}
	if !hasMail || !hasPassword {
		return fmt.Errorf("users table has unsupported schema")
	}

	_, err = db.Exec(`
		ALTER TABLE users RENAME COLUMN mail TO email;
		ALTER TABLE users RENAME COLUMN password TO password_hash;
	`)
	return err
}

func userTableColumns(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns[strings.ToLower(name)] = true
	}
	return columns, rows.Err()
}
