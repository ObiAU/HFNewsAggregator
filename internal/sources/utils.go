package sources

import (
	"crypto/sha256"
	"fmt"
)

func generateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
