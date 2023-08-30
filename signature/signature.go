// Package signature provides helper functions for handling the signature needs.
package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
)

// Set of error variables.
var (
	ErrNoSignatureMatch = errors.New("no signature match")
)

// Validate verifies the message and signature was produced with the specified
// secret key.
func Validate(message, signature, secretKey []byte) error {
	mac, err := Compute(message, secretKey)
	if err != nil {
		return fmt.Errorf("compute: %w", err)
	}

	if !hmac.Equal(signature, mac) {
		return ErrNoSignatureMatch
	}

	return nil
}

func Compute(message, secretKey []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, secretKey)

	if _, err := mac.Write(message); err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	return mac.Sum(nil), nil
}
