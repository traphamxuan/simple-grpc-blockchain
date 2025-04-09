package utils

import (
	pb "blockchain/proto"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// validateBlock checks the previous hash and difficulty
func ValidateBlock(blk *pb.Block, header *pb.BlockHeader) error {
	// 1. Verify previous block hash

	if !bytes.Equal(blk.GetHeader().GetPrevBlockHash(), header.GetPrevBlockHash()) {
		return fmt.Errorf("previous block hash mismatch")
	}

	// 2. Verify proof-of-work (difficulty)
	calculatedHash, err := CalculateBlockHash(blk)
	if err != nil {
		return err
	}
	if !bytes.Equal(blk.GetHash(), calculatedHash) {
		return fmt.Errorf("invalid block hash")
	}

	if compareBits(calculatedHash, header.GetPrevBlockHash(), header.GetBits()) {
		return nil
	}

	return fmt.Errorf("block hash mismatch")
}

// calculateBlockHash computes the double SHA-256 hash of the block header
func CalculateBlockHash(blk *pb.Block) ([]byte, error) {
	// Serialize header (simplified, assumes little-endian)
	var buf bytes.Buffer

	// Handle BlockHeader fields
	header := blk.GetHeader()
	if header == nil {
		return nil, fmt.Errorf("block header is nil")
	}

	// 1. version (int32) - 4 bytes
	if err := binary.Write(&buf, binary.LittleEndian, header.GetVersion()); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}

	// 2. prevBlockHash (string) - write as raw bytes
	if _, err := buf.Write(header.GetPrevBlockHash()); err != nil {
		return nil, fmt.Errorf("failed to write prevBlockHash: %w", err)
	}

	// 3. merkleRoot (string) - write as raw bytes
	if _, err := buf.Write(header.GetMerkleRoot()); err != nil {
		return nil, fmt.Errorf("failed to write merkleRoot: %w", err)
	}

	// 4. timestamp (int64) - 8 bytes
	if err := binary.Write(&buf, binary.LittleEndian, header.GetTimestamp()); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}

	// 5. bits (uint32) - 4 bytes
	if err := binary.Write(&buf, binary.LittleEndian, header.GetBits()); err != nil {
		return nil, fmt.Errorf("failed to write bits: %w", err)
	}

	// 6. nonce (uint64) - 8 bytes
	if err := binary.Write(&buf, binary.LittleEndian, header.GetNonce()); err != nil {
		return nil, fmt.Errorf("failed to write nonce: %w", err)
	}

	// Handle Block fields
	// 2. height (int64) - 8 bytes
	if err := binary.Write(&buf, binary.LittleEndian, header.GetHeight()); err != nil {
		return nil, fmt.Errorf("failed to write height: %w", err)
	}

	// 3. data (string) - write as raw bytes
	data := []byte(blk.GetData())
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(buf.Bytes())
	return hash[:], nil
}

func compareBits(calculatedHash, preBlkHash []byte, difficulty uint32) bool {
	// Convert difficulty from bits to bytes and remaining bits
	byteLength := int(difficulty / 8)
	bitRemainder := difficulty % 8

	// Compare full bytes first
	if byteLength > 0 && !bytes.Equal(calculatedHash[0:byteLength], preBlkHash[(len(preBlkHash)-byteLength):]) {
		return false
	}

	// Compare remaining bits if any
	if bitRemainder > 0 {
		// Get the last byte to compare
		calcByte := calculatedHash[byteLength]
		prevByte := preBlkHash[len(preBlkHash)-len(calculatedHash)+byteLength]

		// Create a mask for the remaining bits
		mask := byte(0xFF) << (8 - bitRemainder)

		// Compare only the significant bits
		return (calcByte & mask) == (prevByte & mask)
	}

	return true
}
