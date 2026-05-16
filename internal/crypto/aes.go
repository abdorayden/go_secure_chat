// Copyright (c) 2026 abdenour souane. All Rights Reserved.

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidAESKeySize = errors.New("invalid AES key size")
	ErrCiphertextShort   = errors.New("ciphertext too short")
)

func GenerateAESKey(size int) ([]byte, error) {
	switch size {
	case 16, 24, 32:
	default:
		return nil, ErrInvalidAESKeySize
	}
	key := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, key)
	return key, err
}

func EncryptAES(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
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
	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrCiphertextShort
	}
	nonce := ciphertext[:gcm.NonceSize()]
	body := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
}

func EncryptBase64(key, plaintext []byte) (string, error) {
	ciphertext, err := EncryptAES(key, plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptBase64(key []byte, payload string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	return DecryptAES(key, ciphertext)
}
