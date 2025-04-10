package utils

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

func RandomUint64() (uint64, error) {
	b := make([]byte, 8)
	if n, err := rand.Read(b); err != nil || n < 8 {
		return 0, fmt.Errorf("failed to generate random uint64: %w", err)
	}
	return binary.BigEndian.Uint64(b), nil
}
