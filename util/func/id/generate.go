package id

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func Generate() string {
	timestamp := time.Now().Format("20060102150405")
	randomBytes := make([]byte, 8)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return timestamp
	}

	return timestamp + "-" + hex.EncodeToString(randomBytes)
}
