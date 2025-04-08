package utils

import (
	pb "blockchain/proto"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/wire"
)

// validateBlock checks the previous hash and difficulty
func ValidateBlock(header *pb.BlockHeader, prevBlockHash string, newBlockHash string) error {
	// 1. Verify previous block hash
	if header.PrevBlockHash != prevBlockHash {
		return fmt.Errorf("previous block hash mismatch: expected %s, got %s", prevBlockHash, header.PrevBlockHash)
	}

	// 2. Verify proof-of-work (difficulty)
	calculatedHash := CalculateBlockHash(header)
	if calculatedHash != newBlockHash {
		return fmt.Errorf("block hash mismatch: calculated %s, expected %s", calculatedHash, newBlockHash)
	}

	// Check if hash meets the difficulty target
	target := compactToBig(header.Bits)
	hashInt := hashToBig(calculatedHash)
	if hashInt.Cmp(target) > 0 {
		return fmt.Errorf("block hash %s does not meet difficulty target %s", calculatedHash, target.Text(16))
	}

	return nil
}

// calculateBlockHash computes the double SHA-256 hash of the block header
func CalculateBlockHash(header *pb.BlockHeader) string {
	// Serialize header (simplified, assumes little-endian)
	var buf bytes.Buffer
	buf.Write(int32ToBytes(header.Version))
	prevHash, _ := hex.DecodeString(header.PrevBlockHash)
	buf.Write(reverseBytes(prevHash)) // Bitcoin uses little-endian
	merkleRoot, _ := hex.DecodeString(header.MerkleRoot)
	buf.Write(reverseBytes(merkleRoot))
	buf.Write(int64ToBytes(header.Timestamp))
	buf.Write(uint32ToBytes(header.Bits))
	buf.Write(uint64ToBytes(header.Nonce))

	// Double SHA-256
	hash1 := sha256.Sum256(buf.Bytes())
	hash2 := sha256.Sum256(hash1[:])
	return hex.EncodeToString(reverseBytes(hash2[:])) // Reverse for Bitcoin convention
}

// compactToBig converts compact difficulty (bits) to a big.Int target
func compactToBig(bits uint32) *big.Int {
	// Extract exponent and mantissa
	exponent := bits >> 24
	mantissa := bits & 0xFFFFFF

	// Calculate target
	target := big.NewInt(int64(mantissa))
	shift := 8 * (int(exponent) - 3)
	target.Lsh(target, uint(shift))
	return target
}

// hashToBig converts a hex hash to a big.Int
func hashToBig(hash string) *big.Int {
	hashBytes, _ := hex.DecodeString(hash)
	return new(big.Int).SetBytes(reverseBytes(hashBytes))
}

// Helper functions for serialization
func int32ToBytes(i int32) []byte {
	b := make([]byte, 4)
	wire.WriteVarInt(&bytes.Buffer{}, 0, uint64(i)) // Simplified, assumes little-endian
	return b
}

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	wire.WriteVarInt(&bytes.Buffer{}, 0, uint64(i))
	return b
}

func uint32ToBytes(i uint32) []byte {
	b := make([]byte, 4)
	wire.WriteVarInt(&bytes.Buffer{}, 0, uint64(i))
	return b
}

func uint64ToBytes(i uint64) []byte {
	b := make([]byte, 8)
	wire.WriteVarInt(&bytes.Buffer{}, 0, uint64(i))
	return b
}

func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		r[i] = b[len(b)-1-i]
	}
	return r
}
