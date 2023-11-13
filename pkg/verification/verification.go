package verification

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// VerifyEmail takes an email address string and verifies that it meets the standard email format
func VerifyEmail(email string) bool {
	return emailRegex.MatchString(email)
}
