package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	FakeAccountID   = "123456789012"
	FakeAccessKeyID = "AKIAIOSFODNN7EXAMPLE"
	FakeSecretKey   = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	FakeSessionToken = "FakeSessionToken"
	FakeRegion      = "us-east-1"
)

// GenerateMessageID produces a realistic SES message ID.
func GenerateMessageID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-000000", hex.EncodeToString(b))
}

// GenerateARN produces a fake AWS ARN.
func GenerateARN(service, resource string) string {
	return fmt.Sprintf("arn:aws:%s:%s:%s:%s", service, FakeRegion, FakeAccountID, resource)
}

// FakeCredentials returns a set of temporary credentials with the given expiry.
func FakeCredentials(expiry time.Duration) (accessKeyID, secretKey, sessionToken string, expiration time.Time) {
	return "ASIAFAKEKEY" + randomHex(8),
		"fakesecret" + randomHex(20),
		"fakesession" + randomHex(40),
		time.Now().Add(expiry)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
