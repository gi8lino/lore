package auth

import "unicode/utf8"

const (
	// MinimumLocalPasswordCharacters counts Unicode code points, not UTF-8 bytes.
	MinimumLocalPasswordCharacters = 12
	// MaximumLocalPasswordBytes is the input limit enforced by bcrypt.
	MaximumLocalPasswordBytes = 72
)

// LocalPasswordProblem returns a user-facing validation message, or an empty string.
func LocalPasswordProblem(password string) string {
	if !utf8.ValidString(password) {
		return "Use valid UTF-8 characters."
	}
	if utf8.RuneCountInString(password) < MinimumLocalPasswordCharacters {
		return "Use at least 12 characters."
	}
	if len(password) > MaximumLocalPasswordBytes {
		return "Use at most 72 UTF-8 bytes."
	}
	return ""
}

// ValidLocalPassword reports whether a password can be accepted and hashed.
func ValidLocalPassword(password string) bool {
	return LocalPasswordProblem(password) == ""
}
