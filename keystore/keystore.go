// Package keystore implements the auth.KeyLookup interface. This implements
// an in-memory keystore for JWT support.
package keystore

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// PrivateKey represents key information.
type PrivateKey struct {
	PK  *rsa.PrivateKey
	PEM []byte
}

// KeyStore represents an in memory store implementation of the
// KeyLookup interface for use with the auth package.
type KeyStore struct {
	store map[string]PrivateKey
}

// New constructs an empty KeyStore ready for use.
func New() (*KeyStore, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	// Construct a PEM block for the private key.
	privateBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the private key to the private key file.
	var b bytes.Buffer
	if err := pem.Encode(&b, &privateBlock); err != nil {
		return nil, fmt.Errorf("encoding to private key: %w", err)
	}

	pk := PrivateKey{
		PK:  privateKey,
		PEM: b.Bytes(),
	}

	ks := &KeyStore{
		store: map[string]PrivateKey{
			"transient": pk,
		},
	}

	return ks, nil
}

// PrivateKey searches the key store for a given kid and returns the private key.
func (ks *KeyStore) PrivateKey(kid string) (string, error) {
	privateKey, found := ks.store[kid]
	if !found {
		return "", errors.New("kid lookup failed")
	}

	return string(privateKey.PEM), nil
}

// PublicKey searches the key store for a given kid and returns the public key.
func (ks *KeyStore) PublicKey(kid string) (string, error) {
	privateKey, found := ks.store[kid]
	if !found {
		return "", errors.New("kid lookup failed")
	}

	asn1Bytes, err := x509.MarshalPKIXPublicKey(&privateKey.PK.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshaling public key: %w", err)
	}

	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	var b bytes.Buffer
	if err := pem.Encode(&b, &block); err != nil {
		return "", fmt.Errorf("encoding to private file: %w", err)
	}

	return b.String(), nil
}
