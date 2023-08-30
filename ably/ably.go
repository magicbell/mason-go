// Package ably provides a wrapper around the http client to authenticated send
// requests to the Ably.
package ably

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// This provides an optimized client configuration for performant requests
var client = http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// Ably provides support to access the ably web socket api.
type Ably struct {
	client  http.Client
	keyName string
	secret  string
}

// New creates a new ably value with the given API key and optimized.
func New(ablyKey string) (*Ably, error) {
	keyName, secret, found := strings.Cut(ablyKey, ":")
	if !found {
		return nil, fmt.Errorf("invalid ably api key: %s", ablyKey)
	}

	a := Ably{
		client:  client,
		keyName: keyName,
		secret:  secret,
	}

	return &a, nil
}

// ================================================================================

// PublishBatch makes an authenticated POST request to the /messages Ably endpoint
// with a payload of batches and returns the response
func (a *Ably) PublishBatch(ctx context.Context, batches []Batch) error {
	body, err := json.Marshal(batches)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}

	url := "https://rest.ably.io/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("post ably /messages: %w", err)
	}

	req.SetBasicAuth(a.keyName, a.secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("post ably /messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("post ably /messages: %w", err)
	}

	return nil
}

// CreateToken creates a new token request with the given projectID and userID.
func (a *Ably) CreateToken(ctx context.Context, projectID string, userID uuid.UUID) (Token, error) {
	channel := fmt.Sprintf("project:%s:channel:%s", projectID, userID)
	capability := fmt.Sprintf(`{"%s":["*"]}`, channel)

	tp := TokenParams{
		TTL:        24 * 60 * 60 * 1000,
		Capability: capability,
		ClientID:   userID.String(),
		Timestamp:  time.Now().UnixMilli(),
	}

	p := make([]byte, 32/2+1)
	rand.Read(p)

	t := Token{
		KeyName:     a.keyName,
		TokenParams: tp,
		Nonce:       hex.EncodeToString(p)[:32],
	}
	t.sign(a.secret)

	return t, nil
}
