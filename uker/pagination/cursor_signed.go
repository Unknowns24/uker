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

type cursorNoSig struct {
	Version   int               `json:"v"`
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
func EncodeCursorSigned(payload CursorPayload, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("pagination: missing cursor signing secret")
	}
	if payload.Version == 0 {
		payload.Version = 1
	}
	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().Unix()
	}

	core := cursorNoSig{
		Version:   payload.Version,
		Sort:      payload.Sort,
		Filters:   payload.Filters,
		After:     payload.After,
		Before:    payload.Before,
		Timestamp: payload.Timestamp,
	}

	signature, err := signCursorPayload(core, secret)
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
func DecodeCursorSigned(encoded string, secret []byte, ttl time.Duration) (CursorPayload, error) {
	if encoded == "" {
		return CursorPayload{}, ErrInvalidCursor
	}
	if len(secret) == 0 {
		return CursorPayload{}, ErrInvalidCursor
	}

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
		Sort:      payload.Sort,
		Filters:   payload.Filters,
		After:     payload.After,
		Before:    payload.Before,
		Timestamp: payload.Timestamp,
	}

	expected, err := signCursorPayload(core, secret)
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
