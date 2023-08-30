package ably

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// Messages is the struct for Ably's Messages object.
type Messages struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

// Batch is the struct for Ably's Batch object.
type Batch struct {
	Messages Messages `json:"messages"`
	Channels []string `json:"channels"`
}

// TokenParams contains token params to be sent to ably to get auth token.
type TokenParams struct {
	// TTL is a requested time to live for the token in milliseconds. If the token request
	// is successful, the TTL of the returned token will be less than or equal
	// to this value depending on application settings and the attributes
	// of the issuing key.
	// The default is 60 minutes (RSA9e, TK2a).
	TTL int64

	// Capability represents encoded channel access rights associated with this Ably Token.
	// The capabilities value is a JSON-encoded representation of the resource paths and associated operations.
	// Read more about capabilities in the [capabilities docs].
	// default '{"*":["*"]}' (RSA9f, TK2b)
	//
	// [capabilities docs]: https://ably.com/docs/core-features/authentication/#capabilities-explained
	Capability string

	// ClientID is used for identifying this client when publishing messages or for presence purposes.
	// The clientId can be any non-empty string, except it cannot contain a *. This option is primarily intended
	// to be used in situations where the library is instantiated with a key.
	// Note that a clientId may also be implicit in a token used to instantiate the library.
	// An error is raised if a clientId specified here conflicts with the clientId implicit in the token.
	// Find out more about [identified clients] (TK2c).
	//
	// [identified clients]: https://ably.com/docs/core-features/authentication#identified-clients
	ClientID string

	// Timestamp of the token request as milliseconds since the Unix epoch.
	// Timestamps, in conjunction with the nonce, are used to prevent requests from being replayed.
	// timestamp is a "one-time" value, and is valid in a request, but is not validly a member of
	// any default token params such as ClientOptions.defaultTokenParams (RSA9d, Tk2d).
	Timestamp int64
}

// Token contains tokenparams with extra details, sent to ably for getting auth token
type Token struct {
	TokenParams

	// KeyName is the name of the key against which this request is made.
	// The key name is public, whereas the key secret is private (TE2).
	KeyName string

	// Nonce is a cryptographically secure random string of at least 16 characters,
	// used to ensure the TokenRequest cannot be reused (TE2).
	Nonce string

	// MAC is the Message Authentication Code for this request.
	MAC string
}

func (t *Token) sign(secret string) {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintln(mac, t.KeyName)
	fmt.Fprintln(mac, t.TTL)
	fmt.Fprintln(mac, t.Capability)
	fmt.Fprintln(mac, t.ClientID)
	fmt.Fprintln(mac, t.Timestamp)
	fmt.Fprintln(mac, t.Nonce)

	t.MAC = base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
