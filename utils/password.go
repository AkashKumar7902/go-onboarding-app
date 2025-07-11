package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword takes a plaintext password and returns its bcrypt hash.
// The cost parameter (14) determines how much computational effort is used.
// A higher cost is more secure but slower. 14 is a good modern default.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a stored bcrypt hash.
// It returns true if the password matches the hash, and false otherwise.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // err is nil on success
}
