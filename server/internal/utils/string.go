package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func GenerateOTP(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be a positive integer")
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	modulus := uint64(math.Pow(10, float64(length)))
	n := binary.BigEndian.Uint64(b[:]) % modulus
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n), nil
}

func HashWithSHA256(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
