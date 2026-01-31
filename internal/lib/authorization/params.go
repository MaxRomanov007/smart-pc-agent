package authorization

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
)

// randomBytesSize defines the number of random bytes used for state and verifier generation
const randomBytesSize = 32

// pkceAuthParams holds parameters required for PKCE (Proof Key for Code Exchange) flow
// PKCE enhances security for public OAuth2 clients by preventing authorization code interception
type pkceAuthParams struct {
	port      int    // Port for callback server
	state     string // CSRF protection token
	verifier  string // PKCE code verifier (random secret)
	challenge string // PKCE code challenge (hashed verifier)
}

// generatePKCEParams generates all required parameters for PKCE OAuth2 flow
// Returns a free port, random state, code verifier, and code challenge
func generatePKCEParams() (*pkceAuthParams, error) {
	const op = "lib.authorization.params.generatePKCEParams"

	port, err := getFreePort()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get free port: %w", op, err)
	}

	state, err := generateRandomString()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate state: %w", op, err)
	}

	verifier, err := generateRandomString()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to generate verifier: %w", op, err)
	}

	challenge := generateChallenge(verifier)

	return &pkceAuthParams{
		port:      port,
		state:     state,
		verifier:  verifier,
		challenge: challenge,
	}, nil
}

// generateRandomString generates a cryptographically secure random string
// Used for state and code verifier parameters in OAuth2 flow
// Returns a base64 URL-encoded string of 32 random bytes
func generateRandomString() (string, error) {
	const op = "lib.authorization.params.generateRandomString"

	data := make([]byte, randomBytesSize)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("%s: failed to read buffer by rand: %w", op, err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// generateChallenge creates a SHA256 hash of the verifier for PKCE
// The challenge is sent to the authorization server, while verifier is kept secret
// Returns base64 URL-encoded SHA256 hash of the verifier
func generateChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// getFreePort finds an available TCP port on localhost
// Uses net.ListenTCP on port 0 to let the OS assign a free port
// Returns the assigned port number or an error if no port is available
func getFreePort() (int, error) {
	const op = "lib.authorization.config.getFreePort"

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("%s: failed to resolve tcp address: %w", op, err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to listen tcp: %w", op, err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
