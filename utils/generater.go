package utils

import (
	"crypto/rand"
	"fmt"
)

func GenerateOTP() string {
	// Generate a 4-digit OTP
	var number [1]byte
	rand.Read(number[:])
	return fmt.Sprintf("%04d", int(number[0])%10000)
}

func GenerateUUID() string {
	// Generate a UUID
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
