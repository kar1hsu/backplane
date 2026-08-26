package utils

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password {
		t.Fatal("password hash must not contain the plain password")
	}
	if !CheckPassword(password, hash) {
		t.Fatal("CheckPassword() rejected the correct password")
	}
	if CheckPassword("wrong password", hash) {
		t.Fatal("CheckPassword() accepted an incorrect password")
	}

	secondHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword() error = %v", err)
	}
	if secondHash == hash {
		t.Fatal("bcrypt hashes should use distinct salts")
	}
}
