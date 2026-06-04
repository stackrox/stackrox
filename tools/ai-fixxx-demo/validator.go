package demo

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail checks whether an email address is valid.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// ValidatePassword checks password strength requirements.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, ch := range password {
		if unicode.IsUpper(ch) {
			hasUpper = true
		}
		if unicode.IsLower(ch) {
			hasLower = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain uppercase, lowercase, and digit")
	}
	return nil
}

// SanitizeInput removes leading/trailing whitespace and collapses internal spaces.
func SanitizeInput(input string) string {
	trimmed := strings.TrimSpace(input)
	words := strings.Fields(trimmed)
	return strings.Join(words, " ")
}

// ValidateUsername checks that a username meets naming requirements.
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("username must be between 3 and 32 characters")
	}
	for i, ch := range username {
		if i == 0 && !unicode.IsLetter(ch) {
			return fmt.Errorf("username must start with a letter")
		}
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return fmt.Errorf("username can only contain letters, digits, and underscores")
		}
	}
	return nil
}

// ParseTags splits a comma-separated tag string and normalizes each tag.
func ParseTags(input string) []string {
	if input == "" {
		return nil
	}
	raw := strings.Split(input, ",")
	tags := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		t = strings.ToLower(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
