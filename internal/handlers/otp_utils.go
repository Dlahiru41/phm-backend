package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// hashOTP hashes an OTP code using SHA256
func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(sum[:])
}

// generateOTPCode generates a random 6-digit OTP
func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// maskPhone masks all but the last 4 digits of a phone number
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

// normalizePhone validates and normalizes a phone number
func normalizePhone(value string) (string, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "", fmt.Errorf("empty")
	}
	if strings.HasPrefix(clean, "+") {
		clean = "+" + strings.ReplaceAll(clean[1:], " ", "")
	} else {
		clean = strings.ReplaceAll(clean, " ", "")
	}
	if len(clean) < 10 || len(clean) > 16 {
		return "", fmt.Errorf("invalid length")
	}
	for i, ch := range clean {
		if i == 0 && ch == '+' {
			continue
		}
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("invalid chars")
		}
	}
	if clean[0] != '+' {
		clean = "+" + clean
	}
	return clean, nil
}
