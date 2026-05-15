package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
)

func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, bits)
}

func MarshalPublicKey(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(der), nil
}

func ParsePublicKey(raw string) (*rsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	return key.(*rsa.PublicKey), nil
}

func EncryptRSAOAEP(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	hash := sha256.New()
	return rsa.EncryptOAEP(hash, rand.Reader, pub, plaintext, nil)
}

func DecryptRSAOAEP(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	hash := sha256.New()
	return rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, nil)
}
