package enc

import (
	"crypto/rsa" // for more code example
)

type RSAPublicKey struct {
	rsa.PublicKey
}

type RSAPrivateKey struct {
	rsa.PrivateKey
}
