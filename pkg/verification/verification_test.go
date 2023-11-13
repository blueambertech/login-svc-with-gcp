package verification

import "testing"

func TestVerifyEmail(t *testing.T) {
	validEmail := "hello@test.com"
	invalidEmails := []string{
		"fred",
		"fred@",
		"fred@test",
		"fred@test.",
		"@test.com",
		"test.com",
		"test.",
	}
	if VerifyEmail(validEmail) == false {
		t.Error("incorrectly validated", validEmail, "as false")
		return
	}
	for _, email := range invalidEmails {
		if VerifyEmail(email) == true {
			t.Error("incorrectly validated", email, "as true")
		}
	}
}
