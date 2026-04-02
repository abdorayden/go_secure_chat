package enc

import (
	"bytes"
	"math/big"
	"testing"
)

func TestModExp(t *testing.T) {
	tests := []struct {
		base     int64
		exp      int64
		mod      int64
		expected int64
	}{
		{2, 3, 10, 8},
		{5, 0, 10, 1},
		{2, 10, 1000, 24},
	}

	for _, tt := range tests {
		result := modExp(
			big.NewInt(tt.base),
			big.NewInt(tt.exp),
			big.NewInt(tt.mod),
		)
		if result.Int64() != tt.expected {
			t.Errorf("modExp(%d, %d, %d) = %d, want %d", tt.base, tt.exp, tt.mod, result.Int64(), tt.expected)
		}
	}
}

func TestGCD(t *testing.T) {
	tests := []struct {
		a        int64
		b        int64
		expected int64
	}{
		{12, 8, 4},
		{54, 24, 6},
		{17, 13, 1},
		{100, 25, 25},
	}

	for _, tt := range tests {
		result := gcd(big.NewInt(tt.a), big.NewInt(tt.b))
		if result.Int64() != tt.expected {
			t.Errorf("gcd(%d, %d) = %d, want %d", tt.a, tt.b, result.Int64(), tt.expected)
		}
	}
}

func TestModInverse(t *testing.T) {
	tests := []struct {
		a        int64
		m        int64
		expected int64
	}{
		{3, 11, 4},
		{7, 26, 15},
	}

	for _, tt := range tests {
		result, err := modInverse(big.NewInt(tt.a), big.NewInt(tt.m))
		if err != nil {
			t.Errorf("modInverse(%d, %d) returned error: %v", tt.a, tt.m, err)
			continue
		}
		if result.Int64() != tt.expected {
			t.Errorf("modInverse(%d, %d) = %d, want %d", tt.a, tt.m, result.Int64(), tt.expected)
		}
	}
}

func TestMillerRabin(t *testing.T) {
	primes := []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47}
	composites := []int64{4, 6, 8, 9, 10, 12, 14, 15, 16, 18, 20, 21, 22}

	for _, p := range primes {
		if !millerRabin(big.NewInt(p), 20) {
			t.Errorf("millerRabin(%d) = false, want true (is prime)", p)
		}
	}

	for _, c := range composites {
		if millerRabin(big.NewInt(c), 20) {
			t.Errorf("millerRabin(%d) = true, want false (is composite)", c)
		}
	}
}

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair(512)
	if err != nil {
		t.Fatalf("GenerateKeyPair(512) returned error: %v", err)
	}

	if priv == nil || pub == nil {
		t.Fatal("GenerateKeyPair returned nil key")
	}

	if priv.Size() != 64 {
		t.Errorf("private key size = %d, want 64", priv.Size())
	}
	if pub.Size() != 64 {
		t.Errorf("public key size = %d, want 64", pub.Size())
	}
}

func TestEncryptDecrypt(t *testing.T) {
	priv, pub, err := GenerateKeyPair(1024)
	if err != nil {
		t.Fatalf("GenerateKeyPair(1024) returned error: %v", err)
	}

	testMessages := []string{
		"Hello, World!",
		"Test message",
		"1234567890",
		"",
		"Unicode: 你好世界",
	}

	for _, msg := range testMessages {
		if msg == "" {
			continue
		}

		ciphertext, err := Encrypt(pub, []byte(msg))
		if err != nil {
			t.Errorf("Encrypt(%q) returned error: %v", msg, err)
			continue
		}

		plaintext, err := Decrypt(priv, ciphertext)
		if err != nil {
			t.Errorf("Decrypt() returned error: %v", err)
			continue
		}

		if !bytes.Equal(plaintext, []byte(msg)) {
			t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", msg, plaintext, msg)
		}
	}
}

func TestKeyMarshalUnmarshal(t *testing.T) {
	priv, pub, err := GenerateKeyPair(512)
	if err != nil {
		t.Fatalf("GenerateKeyPair(512) returned error: %v", err)
	}

	pubStr := pub.Marshal()
	unmarshaledPub, err := UnmarshalPublicKey(pubStr)
	if err != nil {
		t.Fatalf("UnmarshalPublicKey failed: %v", err)
	}

	if pub.N.Cmp(unmarshaledPub.N) != 0 {
		t.Errorf("public key N mismatch after marshal/unmarshal")
	}
	if pub.E != unmarshaledPub.E {
		t.Errorf("public key E mismatch after marshal/unmarshal")
	}

	privStr := MarshalPrivateKey(priv)
	unmarshaledPriv, err := UnmarshalPrivateKey(privStr)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKey failed: %v", err)
	}

	if priv.D.Cmp(unmarshaledPriv.D) != 0 {
		t.Errorf("private key D mismatch after marshal/unmarshal")
	}
}

func TestEncryptDifferentPublicKeys(t *testing.T) {
	_, pub1, _ := GenerateKeyPair(512)
	_, pub2, _ := GenerateKeyPair(512)

	ciphertext1, err := Encrypt(pub1, []byte("test"))
	if err != nil {
		t.Fatalf("Encrypt with pub1 failed: %v", err)
	}

	_, err = Encrypt(pub2, ciphertext1)
	if err == nil {
		t.Error("Encrypt should fail when ciphertext is used as message")
	}
}

func TestEncryptDecryptConsistency(t *testing.T) {
	priv, pub, err := GenerateKeyPair(1024)
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	for i := 0; i < 10; i++ {
		msg := []byte("Consistency test message")
		ciphertext, err := Encrypt(pub, msg)
		if err != nil {
			t.Errorf("Iteration %d: Encrypt failed: %v", i, err)
			continue
		}

		plaintext, err := Decrypt(priv, ciphertext)
		if err != nil {
			t.Errorf("Iteration %d: Decrypt failed: %v", i, err)
			continue
		}

		if !bytes.Equal(plaintext, msg) {
			t.Errorf("Iteration %d: mismatch", i)
		}
	}
}
