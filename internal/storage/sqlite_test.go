package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DBModel {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	model, err := Open(dbPath, filepath.Join("..", "..", "migrations", "001_users.sql"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return model
}

func TestCreateAndAuthenticateUser(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := db.CreateUser(ctx, "alice_1", "alice@example.com", "supersecret"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	user, err := db.AuthenticateUser(ctx, "alice@example.com", "supersecret")
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	if user.Username != "alice_1" {
		t.Fatalf("unexpected username: %s", user.Username)
	}
}

func TestDuplicateUsers(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	if err := db.CreateUser(ctx, "alice_1", "alice@example.com", "supersecret"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := db.CreateUser(ctx, "alice_2", "alice@example.com", "supersecret"); err != ErrUserExists {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestSaveAndLoadMessageHistory(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()
	err := db.SaveMessageHistory(ctx, "msg-1", "alice", "ciphertext-1", map[string]string{
		"alice": "wrapped-a",
		"bob":   "wrapped-b",
	}, 1710000000)
	if err != nil {
		t.Fatalf("SaveMessageHistory: %v", err)
	}

	records, err := db.LoadMessageHistoryForUser(ctx, "bob", 50)
	if err != nil {
		t.Fatalf("LoadMessageHistoryForUser: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].SenderUsername != "alice" || records[0].EncryptedKey != "wrapped-b" {
		t.Fatalf("unexpected record: %#v", records[0])
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
