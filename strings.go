package uker

import "strings"

// Global interface
type str interface {
	Contains(str string, chars []string) bool
	HasNoValidChars(str string) bool
}

// Local struct to be implmented
type str_implementation struct{}

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
	invalidChars := []string{"*", "/", "\\", "=", ">", "<", ":", ";", "¿", "?", "\"", "|", "!", "¡", "º"}

	for _, char := range invalidChars {
		if strings.Contains(str, char) {
			return true
		}
	}

	return false
}
