package fn

import "strings"

var invalidChars = [...]string{"*", "/", "\\", "=", ">", "<", ":", ";", "¿", "?", "\"", "|", "!", "¡", "º"}

// Contains reports whether the input string contains at least one of the provided characters.
func Contains(value string, chars []string) bool {
	for _, char := range chars {
		if strings.Contains(value, char) {
			return true
		}
	}

	return false
}

// HasNoValidChars reports whether the string contains any of the characters deemed invalid by the package.
func HasNoValidChars(value string) bool {
	for _, char := range invalidChars {
		if strings.Contains(value, char) {
			return true
		}
	}

	return false
}

// Sanitize removes invalid characters and converts spaces to underscores.
func Sanitize(value string) string {
	parsed := value
	for _, char := range invalidChars {
		parsed = strings.ReplaceAll(parsed, char, "")
	}

	parsed = strings.ToLower(strings.ReplaceAll(parsed, " ", "_"))
	return parsed
}

// SplitByUpperCase splits the provided string using uppercase transitions.
func SplitByUpperCase(value string) []string {
	var (
		words   []string
		current string
	)

	for i, char := range value {
		if i > 0 && char >= 'A' && char <= 'Z' {
			words = append(words, current)
			current = string(char)
			continue
		}

		current += string(char)
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}
