package app

import (
	pb "blockchain/proto"
	"blockchain/utils"
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/types/known/emptypb"
)

type BlockchainServer struct {
	pb.UnimplementedBlockchainServer

	mu             *sync.RWMutex
	blocks         []*pb.Block
	requirement    pb.NewBlockRequirementsResponse
	beforeNewBlock func(context.Context, *pb.Block) error
	afterNewBlock  func(context.Context, *pb.Block) error
}

func (s *BlockchainServer) SetRequirement(ctx context.Context, height int64, hash string, diff int32) *pb.NewBlockRequirementsResponse {
	s.requirement = pb.NewBlockRequirementsResponse{
		NextBlockHeight:  height,
		CurrentBlockHash: hash,
		Difficulty:       diff,
	}
	return &s.requirement
}

func (s *BlockchainServer) GetNewBlockRequirement(ctx context.Context, _ *emptypb.Empty) (*pb.NewBlockRequirementsResponse, error) {
	return &s.requirement, nil
}

func (s *BlockchainServer) AddNewBlock(ctx context.Context, block *pb.Block) (*pb.Block, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify block height
	if block.Height != int64(len(s.blocks)+1) {
		return nil, fmt.Errorf("invalid block height")
	}

	// Verify previous hash
	if len(s.blocks) > 0 {
		lastBlock := s.blocks[len(s.blocks)-1]
		if block.Header.PrevBlockHash != lastBlock.Hash {
			return nil, fmt.Errorf("invalid previous hash")
		}
	} else if block.Header.PrevBlockHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return nil, fmt.Errorf("invalid genesis block previous hash")
	}

	// Verify block hash meets difficulty requirement
	if err := utils.ValidateBlock(block.Header, s.requirement.CurrentBlockHash, block.Hash); err != nil {
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

	if req.Height < 0 || int(req.Height) >= len(s.blocks) {
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
		requirement: pb.NewBlockRequirementsResponse{
			Difficulty: -1,
		},
	}
	if len(events) > 1 {
		s.beforeNewBlock = events[0]
	}
	if len(events) > 2 {
		s.afterNewBlock = events[1]
	}
	return s
}
