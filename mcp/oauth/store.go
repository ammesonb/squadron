// Package oauth is the squadron side of the MCP OAuth 2.1 flow. Tokens are
// process-local until Command Center connection auth owns their persistence.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
)

const (
	runtimeKeyPrefix = "oauth:"
	tokenKeySuffix   = ":token"
	clientKeySuffix  = ":client"
)

// ClientCredentials is what DCR hands back; cached so relogin skips
// registration.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// tokenMu serializes concurrent reads and writes of the OAuth key space,
// because refreshes can race logins.
var tokenMu sync.Mutex
var tokenEntries = make(map[string]string)

// RuntimeTokenStore implements transport.TokenStore against process memory.
type RuntimeTokenStore struct {
	name string
}

func NewRuntimeTokenStore(name string) *RuntimeTokenStore {
	return &RuntimeTokenStore{name: name}
}

func (s *RuntimeTokenStore) GetToken(ctx context.Context) (*transport.Token, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tokenMu.Lock()
	raw, ok := tokenEntries[tokenKeyFor(s.name)]
	tokenMu.Unlock()
	if !ok {
		// Missing tokens surface as needs-login rather than fatal errors.
		return nil, transport.ErrNoToken
	}

	var tok transport.Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return nil, fmt.Errorf("oauth %q: decoding stored token: %w", s.name, err)
	}
	return &tok, nil
}

func (s *RuntimeTokenStore) SaveToken(ctx context.Context, tok *transport.Token) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tok == nil {
		return errors.New("oauth: SaveToken called with nil token")
	}

	// Some mcp-go paths set ExpiresIn but leave ExpiresAt zero — compute it
	// so readers (status, future health check) don't need to.
	if tok.ExpiresAt.IsZero() && tok.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}

	blob, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("oauth %q: encoding token: %w", s.name, err)
	}

	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenEntries[tokenKeyFor(s.name)] = string(blob)
	return nil
}

// DeleteToken wipes the stored token but preserves ClientCredentials so the
// next login can skip DCR. Idempotent.
func DeleteToken(name string) error {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	delete(tokenEntries, tokenKeyFor(name))
	return nil
}

// LoadClientCredentials returns (nil, nil) if none are stored.
func LoadClientCredentials(name string) (*ClientCredentials, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	raw, ok := tokenEntries[clientKeyFor(name)]
	if !ok {
		return nil, nil //nolint:nilerr // missing is a normal state
	}
	var creds ClientCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("oauth %q: decoding client credentials: %w", name, err)
	}
	return &creds, nil
}

func SaveClientCredentials(name string, creds ClientCredentials) error {
	blob, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("oauth %q: encoding client credentials: %w", name, err)
	}
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenEntries[clientKeyFor(name)] = string(blob)
	return nil
}

func DeleteClientCredentials(name string) error {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	delete(tokenEntries, clientKeyFor(name))
	return nil
}

func HasToken(name string) bool {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	_, ok := tokenEntries[tokenKeyFor(name)]
	return ok
}

func tokenKeyFor(name string) string  { return runtimeKeyPrefix + name + tokenKeySuffix }
func clientKeyFor(name string) string { return runtimeKeyPrefix + name + clientKeySuffix }

// TokenSnapshot is a point-in-time copy of the OAuth key space so bulk
// inspectors can read a stable view while tokens are refreshed.
type TokenSnapshot struct {
	entries map[string]string
}

func LoadTokenSnapshot() (*TokenSnapshot, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	entries := make(map[string]string, len(tokenEntries))
	for key, value := range tokenEntries {
		entries[key] = value
	}
	return &TokenSnapshot{entries: entries}, nil
}

// ClearRuntimeState removes process-local OAuth tokens and client credentials.
// It is used when isolating tests and may be used when a worker disconnects.
func ClearRuntimeState() {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	tokenEntries = make(map[string]string)
}

func (s *TokenSnapshot) HasToken(name string) bool {
	_, ok := s.entries[tokenKeyFor(name)]
	return ok
}

func (s *TokenSnapshot) Token(name string) (*transport.Token, error) {
	raw, ok := s.entries[tokenKeyFor(name)]
	if !ok {
		return nil, transport.ErrNoToken
	}
	var tok transport.Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return nil, fmt.Errorf("oauth %q: decoding stored token: %w", name, err)
	}
	return &tok, nil
}
