package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashData(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func HashString(s string) string {
	return HashData([]byte(s))
}
