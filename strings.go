package uker

import "strings"

// Global interface
type str interface {
	// Check if string contains chars
	//
	// @param str string: String to validate.
	//
	// @param chars []string: Chars to search.
	//
	// @return bool: if string contains at least one of the chars will return true
	Contains(str string, chars []string) bool

	// Remove invalid chars from string
	//
	// @param str string: String to be sanitized.
	//
	// @return string: sanitized string
	SanitizeString(str string) string

	// Check if string contains invalid chars
	//
	// @param str string: String to validate.
	//
	// @return bool: if string contains at least one of the invalid chars will return true
	HasNoValidChars(str string) bool
}

// Local struct to be implmented
type str_implementation struct{}

// local variables
var invalidChars = [...]string{"*", "/", "\\", "=", ">", "<", ":", ";", "¿", "?", "\"", "|", "!", "¡", "º"}

// External contructor
func Str() str {
	return &str_implementation{}
}

func (s *str_implementation) Contains(str string, chars []string) bool {
	for _, char := range chars {
		if strings.Contains(str, char) {
			return true
		}
	}

	return false
}

func (s *str_implementation) HasNoValidChars(str string) bool {
	// Check if has no valid characters
	for _, char := range invalidChars {
		if strings.Contains(str, char) {
			return true
		}
	}

	return false
}

func (s *str_implementation) SanitizeString(str string) string {
	parsedStr := str

	// Remove all invalid characters
	for _, char := range invalidChars {
		if strings.Contains(str, char) {
			parsedStr = strings.ReplaceAll(parsedStr, char, "")
		}
	}

	// Remove all spaces
	parsedStr = strings.ToLower(strings.ReplaceAll(parsedStr, " ", "_"))

	return parsedStr
}
