package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

const (
	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	withSpecial  = alphanumeric + "!#$%&*()-_=+[]{}:?"
)

// Generate produces a random secret value for the given generator kind.
func Generate(kind GeneratorKind) (string, error) {
	switch kind {
	case GenJWTSecret:
		return randomString(64, alphanumeric)
	case GenEncryptionKey:
		// 32 random bytes, hex-encoded = 64-char hex string
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		return hex.EncodeToString(b), nil
	case GenPassword:
		return randomString(32, withSpecial)
	case GenAPIKey:
		return randomString(48, alphanumeric)
	case GenCookieSecret:
		return randomString(32, alphanumeric)
	default:
		return randomString(32, alphanumeric)
	}
}

func randomString(length int, charset string) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
