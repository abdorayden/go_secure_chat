package enc

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidAESKeySize  error = errors.New("invalid AES key size: must be 16, 24, or 32 bytes")
	ErrCiphertextTooShort error = errors.New("ciphertext too short")
)

func GenerateAESKey(size int) ([]byte, error) {
	switch size {
	case 16, 24, 32:
		key := make([]byte, size)
		_, err := io.ReadFull(cryptorand.Reader, key)
		if err != nil {
			return nil, err
		}
		return key, nil
	default:
		return nil, ErrInvalidAESKeySize
	}
}

func EncryptAES(key, msg []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, msg, nil)
	out := make([]byte, 0, len(nonce)+len(ciphertext))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// base64 implementation
func EncryptAESBase64(key, msg []byte) (string, error) {
	ciphertext, err := EncryptAES(key, msg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptAESBase64(key []byte, cipherTextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return nil, err
	}
	return DecryptAES(key, ciphertext)
}
