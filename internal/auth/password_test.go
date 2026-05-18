package auth

import "testing"

func TestPasswordHashVerifiesOriginalPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected hash to verify original password")
	}

	if VerifyPassword(hash, "wrong password") {
		t.Fatal("expected hash to reject wrong password")
	}
}
