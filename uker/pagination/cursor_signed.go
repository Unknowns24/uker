package pagination

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/unknowns24/uker/internal/base64url"
)

const signingContextNamespace = "github.com/unknowns24/uker/pagination/signing-context/v1"

type signingConfig struct {
	context string
}

// SigningOption configures signed cursor generation and verification.
type SigningOption func(*signingConfig)

// WithSigningContext binds a signed cursor to opaque application-provided context.
// The exact same context must be supplied when generating and verifying the cursor.
// An empty context preserves the legacy unscoped signing behaviour.
func WithSigningContext(context string) SigningOption {
	return func(cfg *signingConfig) {
		cfg.context = context
	}
}

func newSigningConfig(opts ...SigningOption) signingConfig {
	cfg := signingConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func deriveCursorSigningKey(secret []byte, context string) []byte {
	if context == "" {
		return secret
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingContextNamespace))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(context))
	return mac.Sum(nil)
}

type cursorNoSig struct {
	Version   int               `json:"v"`
	Limit     int               `json:"limit,omitempty"`
	Sort      []SortExpression  `json:"sort,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
	After     map[string]string `json:"after,omitempty"`
	Before    map[string]string `json:"before,omitempty"`
	Timestamp int64             `json:"ts,omitempty"`
}

func signCursorPayload(payload cursorNoSig, secret []byte) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// EncodeCursorSigned serialises the cursor payload and appends an HMAC signature using the provided secret.
// Pass WithSigningContext to bind the signature to external application context.
func EncodeCursorSigned(payload CursorPayload, secret []byte, opts ...SigningOption) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("pagination: missing cursor signing secret")
	}
	cfg := newSigningConfig(opts...)
	effectiveKey := deriveCursorSigningKey(secret, cfg.context)
	if payload.Version == 0 {
		payload.Version = 1
	}
	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().Unix()
	}

	core := cursorNoSig{
		Version:   payload.Version,
		Limit:     payload.Limit,
		Sort:      payload.Sort,
		Filters:   payload.Filters,
		After:     payload.After,
		Before:    payload.Before,
		Timestamp: payload.Timestamp,
	}

	signature, err := signCursorPayload(core, effectiveKey)
	if err != nil {
		return "", err
	}
	payload.Signature = signature

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return base64url.Encode(string(raw)), nil
}

// DecodeCursorSigned verifies the cursor signature and TTL before returning the payload.
// A scoped cursor requires the exact WithSigningContext value used when it was encoded.
func DecodeCursorSigned(encoded string, secret []byte, ttl time.Duration, opts ...SigningOption) (CursorPayload, error) {
	if encoded == "" {
		return CursorPayload{}, ErrInvalidCursor
	}
	if len(secret) == 0 {
		return CursorPayload{}, ErrInvalidCursor
	}
	cfg := newSigningConfig(opts...)
	effectiveKey := deriveCursorSigningKey(secret, cfg.context)

	decoded, err := base64url.Decode(encoded)
	if err != nil {
		return CursorPayload{}, ErrInvalidCursor
	}

	var payload CursorPayload
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return CursorPayload{}, ErrInvalidCursor
	}

	if payload.Version <= 0 || payload.Signature == "" {
		return CursorPayload{}, ErrInvalidCursor
	}

	core := cursorNoSig{
		Version:   payload.Version,
		Limit:     payload.Limit,
		Sort:      payload.Sort,
		Filters:   payload.Filters,
		After:     payload.After,
		Before:    payload.Before,
		Timestamp: payload.Timestamp,
	}

	expected, err := signCursorPayload(core, effectiveKey)
	if err != nil {
		return CursorPayload{}, err
	}

	providedSig, err := base64.RawURLEncoding.DecodeString(payload.Signature)
	if err != nil {
		return CursorPayload{}, ErrInvalidCursor
	}
	expectedSig, err := base64.RawURLEncoding.DecodeString(expected)
	if err != nil {
		return CursorPayload{}, ErrInvalidCursor
	}
	if !hmac.Equal(providedSig, expectedSig) {
		return CursorPayload{}, ErrInvalidCursor
	}

	if ttl > 0 {
		expiresAt := time.Unix(payload.Timestamp, 0).Add(ttl)
		if time.Now().After(expiresAt) {
			return CursorPayload{}, ErrCursorExpired
		}
	}

	return payload, nil
}
