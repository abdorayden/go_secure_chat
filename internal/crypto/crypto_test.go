package crypto

import (
	"bytes"
	"testing"
)

func TestAESEncryptDecrypt(t *testing.T) {
	key, err := GenerateAESKey(32)
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}
	ciphertext, err := EncryptAES(key, []byte("hello"))
	if err != nil {
		t.Fatalf("EncryptAES: %v", err)
	}
	plaintext, err := DecryptAES(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAES: %v", err)
	}
	if !bytes.Equal(plaintext, []byte("hello")) {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}

func TestRSAHandshake(t *testing.T) {
	priv, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair: %v", err)
	}
	key, err := GenerateAESKey(32)
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}
	ciphertext, err := EncryptRSAOAEP(&priv.PublicKey, key)
	if err != nil {
		t.Fatalf("EncryptRSAOAEP: %v", err)
	}
	plaintext, err := DecryptRSAOAEP(priv, ciphertext)
	if err != nil {
		t.Fatalf("DecryptRSAOAEP: %v", err)
	}
	if !bytes.Equal(plaintext, key) {
		t.Fatal("decrypted handshake key mismatch")
	}
}
