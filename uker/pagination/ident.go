package pagination

import (
	"errors"
	"regexp"
)

var identRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// AllowedColumns enables opt-in whitelisting of identifiers. When populated, only
// identifiers present in the map will be accepted by safeIdent. Leaving it nil or
// empty disables the whitelist and relies solely on the regular expression check.
var AllowedColumns map[string]struct{}

// safeIdent validates that the provided identifier is free of SQL meta characters and
// optionally belongs to the AllowedColumns whitelist. It returns ErrInvalidIdentifier
// when the value cannot be used safely in a query.
func safeIdent(value string) (string, error) {
	if value == "" {
		return "", ErrInvalidIdentifier
	}
	if !identRe.MatchString(value) {
		return "", ErrInvalidIdentifier
	}

	if len(AllowedColumns) > 0 {
		if _, ok := AllowedColumns[value]; !ok {
			return "", ErrInvalidIdentifier
		}
	}

	return value, nil
}

// requireIdent acts like safeIdent but converts the sentinel error into the provided one.
func requireIdent(value string, errToWrap error) (string, error) {
	identifier, err := safeIdent(value)
	if err != nil {
		if errors.Is(err, ErrInvalidIdentifier) {
			return "", errToWrap
		}
		return "", err
	}
	return identifier, nil
}
