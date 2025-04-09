package service

import (
	pb "blockchain/proto"
	"blockchain/utils"
	"bytes"
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/types/known/emptypb"
)

var ZERO_HASH = make([]byte, 32, 32)

type BlockchainServer struct {
	pb.UnimplementedBlockchainServer

	mu             *sync.RWMutex
	blocks         []*pb.Block
	header         *pb.BlockHeader
	beforeNewBlock func(context.Context, *pb.Block) error
	afterNewBlock  func(context.Context, *pb.Block) error
}

func (s *BlockchainServer) SetNewBlockHeader(height uint64, hash []byte, diff uint32) *pb.BlockHeader {
	if len(hash) == 0 {
		hash = ZERO_HASH
	}
	s.header = &pb.BlockHeader{
		Height:        height,
		PrevBlockHash: hash,
		Bits:          diff,
	}
	return s.header
}

func (s *BlockchainServer) GetNewBlockHeader() *pb.BlockHeader {
	return s.header
}

func (s *BlockchainServer) AddNewBlock(ctx context.Context, block *pb.Block) (*pb.Block, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Verify block height
	if block.GetHeader().GetHeight() != uint64(len(s.blocks)+1) {
		return nil, fmt.Errorf("invalid block height")
	}

	// Verify previous hash
	if len(s.blocks) > 0 {
		lastBlock := s.blocks[len(s.blocks)-1]
		if !bytes.Equal(block.GetHeader().GetPrevBlockHash(), lastBlock.GetHash()) {
			return nil, fmt.Errorf("invalid previous hash")
		}
	} else if !bytes.Equal(block.Header.PrevBlockHash, ZERO_HASH) {
		return nil, fmt.Errorf("invalid genesis block previous hash")
	}

	// Verify block hash meets difficulty requirement
	if err := utils.ValidateBlock(block, s.header); err != nil {
		return nil, fmt.Errorf("block hash does not meet difficulty requirement: %w", err)
	}
	if s.beforeNewBlock != nil {
		if err := s.beforeNewBlock(ctx, block); err != nil {
			return nil, err
		}
	}

	// Add block to chain
	s.blocks = append(s.blocks, block)

	if s.afterNewBlock != nil {
		if err := s.afterNewBlock(ctx, block); err != nil {
			return nil, err
		}
	}

	return block, nil
}

func (s *BlockchainServer) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if req.Height >= uint64(len(s.blocks)) {
		return nil, fmt.Errorf("block height %d not found", req.Height)
	}

	return s.blocks[req.Height], nil
}

func (s *BlockchainServer) GetHighestBlock(context.Context, *emptypb.Empty) (*pb.Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.blocks) == 0 {
		return nil, fmt.Errorf("blockchain is empty")
	}

	return s.blocks[len(s.blocks)-1], nil
}

func NewBlockNode(events ...func(context.Context, *pb.Block) error) *BlockchainServer {
	s := &BlockchainServer{
		mu: &sync.RWMutex{},
	}
	if len(events) > 0 {
		s.beforeNewBlock = events[0]
	}
	if len(events) > 1 {
		s.afterNewBlock = events[1]
	}
	return s
}
