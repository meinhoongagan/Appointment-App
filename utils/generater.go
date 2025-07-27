package utils

import (
	"crypto/rand"
	"fmt"
)

func GenerateOTP() string {
	type otpResult struct {
		OTP string
	}

	resultChan := make(chan otpResult)

	go func() {
		var result otpResult
		// Generate a 4-digit OTP
		var number [1]byte
		rand.Read(number[:])
		result.OTP = fmt.Sprintf("%04d", int(number[0])%10000)
		resultChan <- result
	}()

	result := <-resultChan
	return result.OTP
}

func GenerateUUID() string {
	type uuidResult struct {
		UUID string
	}

	resultChan := make(chan uuidResult)

	go func() {
		var result uuidResult
		// Generate a UUID
		b := make([]byte, 16)
		rand.Read(b)
		result.UUID = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
		resultChan <- result
	}()

	result := <-resultChan
	return result.UUID
}
