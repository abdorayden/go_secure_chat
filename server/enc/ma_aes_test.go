package enc

import "testing"

func TestGenerateAESKeySizes(t *testing.T) {
	t.Run("valid sizes", func(t *testing.T) {
		for _, size := range []int{16, 24, 32} {
			key, err := GenerateAESKey(size)
			if err != nil {
				t.Fatalf("GenerateAESKey(%d) returned error: %v", size, err)
			}
			if len(key) != size {
				t.Fatalf("GenerateAESKey(%d) length = %d, want %d", size, len(key), size)
			}
		}
	})

	t.Run("invalid size", func(t *testing.T) {
		_, err := GenerateAESKey(20)
		if err == nil {
			t.Fatal("GenerateAESKey(20) expected error, got nil")
		}
	})
}

func TestEncryptDecryptAES(t *testing.T) {
	key, err := GenerateAESKey(32)
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	msg := []byte("hello AES-GCM")
	ciphertext, err := EncryptAES(key, msg)
	if err != nil {
		t.Fatalf("EncryptAES failed: %v", err)
	}

	plaintext, err := DecryptAES(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptAES failed: %v", err)
	}

	if string(plaintext) != string(msg) {
		t.Fatalf("DecryptAES(EncryptAES(msg)) = %q, want %q", plaintext, msg)
	}
}

func TestEncryptDecryptAESBase64(t *testing.T) {
	key, err := GenerateAESKey(16)
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	msg := []byte("base64 test")
	ciphertextB64, err := EncryptAESBase64(key, msg)
	if err != nil {
		t.Fatalf("EncryptAESBase64 failed: %v", err)
	}

	plaintext, err := DecryptAESBase64(key, ciphertextB64)
	if err != nil {
		t.Fatalf("DecryptAESBase64 failed: %v", err)
	}

	if string(plaintext) != string(msg) {
		t.Fatalf("DecryptAESBase64(EncryptAESBase64(msg)) = %q, want %q", plaintext, msg)
	}
}

func TestDecryptAESTamper(t *testing.T) {
	key, err := GenerateAESKey(32)
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	msg := []byte("tamper test")
	ciphertext, err := EncryptAES(key, msg)
	if err != nil {
		t.Fatalf("EncryptAES failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("EncryptAES returned empty ciphertext")
	}

	ciphertext[len(ciphertext)-1] ^= 0x01
	if _, err := DecryptAES(key, ciphertext); err == nil {
		t.Fatal("DecryptAES expected error for tampered ciphertext, got nil")
	}
}

func TestDecryptAESShortCiphertext(t *testing.T) {
	key, err := GenerateAESKey(16)
	if err != nil {
		t.Fatalf("GenerateAESKey failed: %v", err)
	}

	if _, err := DecryptAES(key, []byte{0x01, 0x02}); err == nil {
		t.Fatal("DecryptAES expected error for short ciphertext, got nil")
	}
}
