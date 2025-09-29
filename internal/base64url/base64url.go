package base64url

import (
	"encoding/base64"
	"strings"
)

const prefix = "b64!"

// IsEncoded reports whether the provided string uses the library prefix.
func IsEncoded(value string) bool {
	return strings.HasPrefix(value, prefix)
}

// Decode removes the prefix (when present) and decodes the remaining
// base64 payload into a plain string. When the input is not prefixed it is
// returned as is.
func Decode(value string) (string, error) {
	if !IsEncoded(value) {
		return value, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// Encode encodes the provided value using the same prefix used by Decode.
func Encode(value string) string {
	if value == "" {
		return value
	}

	return prefix + base64.StdEncoding.EncodeToString([]byte(value))
}
