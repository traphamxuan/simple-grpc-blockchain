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

var ZERO_HASH = make([]byte, 32)

type PreBlock struct {
	block *pb.Block
	vote  int
}

type BlockchainServer struct {
	pb.UnimplementedBlockchainServer

	mu            *sync.RWMutex
	blocks        []*pb.Block
	preBlocks     []PreBlock
	header        *pb.BlockHeader
	afterNewBlock func(context.Context, *pb.Block) error
}

func (s *BlockchainServer) SetNewBlockHeader(height uint64, hash []byte, diff uint32) *pb.BlockHeader {
	if len(hash) == 0 {
		hash = ZERO_HASH
	}
	for _, preBlk := range s.preBlocks {
		if bytes.Equal(preBlk.block.GetHash(), hash) {
			s.blocks = append(s.blocks, preBlk.block)
			s.preBlocks = nil
			break
		}
	}
	s.header = &pb.BlockHeader{
		Height:        height,
		PrevBlockHash: hash,
		Bits:          diff,
	}
	return s.header
}

func (s *BlockchainServer) GetNewBlockHeader() *pb.BlockHeader {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.header
}

func (s *BlockchainServer) GetPreBlocks() []PreBlock {
	return s.preBlocks
}

func (s *BlockchainServer) AddNewBlock(ctx context.Context, block *pb.Block) (*pb.Block, error) {
	// Verify block height
	s.mu.RLock()
	if err := s.validateNewBlock(block); err != nil {
		s.mu.RUnlock()
		return nil, err
	}
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateNewBlock(block); err != nil {
		return nil, err
	}

	if block.GetHeader().GetHeight() >= uint64(len(s.blocks)) {
		isRun := false
		for _, preBlk := range s.preBlocks {
			if bytes.Equal(preBlk.block.GetHash(), block.GetHash()) {
				preBlk.vote++
				isRun = true
				break
			}
		}
		if !isRun {
			s.preBlocks = append(s.preBlocks, PreBlock{block, 1})
		}
	}

	if s.afterNewBlock != nil {
		s.mu.Unlock()
		if err := s.afterNewBlock(ctx, block); err != nil {
			fmt.Println("warning: Skip failure in process after new block")
		}
		s.mu.Lock()
	}

	return block, nil
}

func (s *BlockchainServer) addNewBlock(block *pb.Block) (*pb.Block, error) {
	s.mu.RLock()
	if err := s.validateNewBlock(block); err != nil {
		s.mu.RUnlock()
		return nil, err
	}
	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateNewBlock(block); err != nil {
		return nil, err
	}

	if block.GetHeader().GetHeight() >= uint64(len(s.blocks)) {
		isRun := false
		for _, preBlk := range s.preBlocks {
			if bytes.Equal(preBlk.block.GetHash(), block.GetHash()) {
				preBlk.vote++
				isRun = true
				break
			}
		}
		if !isRun {
			s.preBlocks = append(s.preBlocks, PreBlock{block, 1})
		}
	}
	return block, nil
}

func (s *BlockchainServer) validateNewBlock(block *pb.Block) error {
	if block.GetHeader().GetHeight() != s.header.GetHeight() {
		return fmt.Errorf("invalid height: %d vs %d", block.GetHeader().GetHeight(), s.header.GetHeight())
	}

	// Verify block hash meets difficulty requirement
	if err := utils.ValidateBlock(block, s.header); err != nil {
		return fmt.Errorf("block hash does not meet difficulty requirement: %w", err)
	}
	return nil
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

func NewBlockNode(afterNewBlock func(context.Context, *pb.Block) error) *BlockchainServer {
	s := &BlockchainServer{
		mu: &sync.RWMutex{},
	}
	s.afterNewBlock = afterNewBlock
	return s
}
