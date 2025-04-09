package service

import (
	pb "blockchain/proto"
	"blockchain/utils"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Miner struct {
	master      pb.MasterClient
	blockServer *BlockchainServer
	miners      map[string]pb.BlockchainClient
	nodeInfo    pb.NodeInfo
	mu          *sync.RWMutex
}

func NewMiner(masterHost string, masterPort int32, host string, port int32) (*Miner, error) {
	// Connect to master server
	masterKey := fmt.Sprintf("%s:%d", masterHost, masterPort)
	conn, err := grpc.NewClient(masterKey, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}

	return &Miner{
		master: pb.NewMasterClient(conn),
		miners: map[string]pb.BlockchainClient{
			masterKey: pb.NewBlockchainClient(conn),
		},
		nodeInfo: pb.NodeInfo{Host: host, Port: port},
		mu:       &sync.RWMutex{},
	}, nil
}

func (s *Miner) Start(ctx context.Context) (err error) {
	chErr := make(chan error)
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.nodeInfo.Port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		server := grpc.NewServer()

		s.blockServer = NewBlockNode(nil, s.OnNewBlock)
		pb.RegisterBlockchainServer(server, s.blockServer)

		log.Printf("Master server listening at %v", lis.Addr())
		chErr <- server.Serve(lis)
	}()

	select {
	case err = <-chErr:
		return err
	case err = <-s.listenOnNewMiner(ctx):
		return err
	case err = <-s.listenOnNewBlockRequirement(ctx):
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *Miner) listenOnNewMiner(ctx context.Context) chan error {
	chErr := make(chan error)
	go func() {
		var (
			stream   pb.Master_RegisterNodeClient
			nodeInfo pb.NodeInfo
			err      error
		)
		if err := backoff.Retry(func() error {
			stream, err = s.master.RegisterNode(ctx, &s.nodeInfo)
			return err
		}, backoff.NewExponentialBackOff()); err != nil {
			chErr <- err
			return
		}
		for err == nil {
			err = stream.RecvMsg(&nodeInfo)
			if err != nil {
				continue
			}
			if nodeInfo.Host != "" && nodeInfo.Port > 0 {
				nodeKey := fmt.Sprintf("%s:%d", nodeInfo.Host, nodeInfo.Port)
				conn, err := grpc.NewClient(nodeKey, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					fmt.Println("failed to connect to " + nodeKey)
					continue
				}
				s.miners[nodeKey] = pb.NewBlockchainClient(conn)
			}
		}
		chErr <- nil
	}()
	return chErr
}

func (s *Miner) listenOnNewBlockRequirement(ctx context.Context) chan error {
	chErr := make(chan error)
	go func() {
		var (
			stream      pb.Master_RegisterNewBlockRequirementsClient
			requirement pb.NewBlockRequirementsResponse
			err         error
		)
		if err := backoff.Retry(func() error {
			stream, err = s.master.RegisterNewBlockRequirements(ctx, &s.nodeInfo)
			return err
		}, backoff.NewExponentialBackOff()); err != nil {
			chErr <- err
			return
		}
		for err == nil {
			err = stream.RecvMsg(&requirement)
			if err != nil {
				continue
			}
			if requirement.Difficulty >= 0 {
				s.stopMining()
				s.blockServer.SetRequirement(ctx,
					requirement.GetNextBlockHeight(),
					requirement.GetCurrentBlockHash(),
					requirement.GetDifficulty())
				go s.startMining(ctx, &requirement)
			}
		}
		chErr <- nil
	}()
	return chErr
}

func (s *Miner) OnNewBlock(ctx context.Context, blk *pb.Block) error {
	return nil
}

func (s *Miner) startMining(ctx context.Context, requirement *pb.NewBlockRequirementsResponse) {
	for {
		if requirement == nil || requirement.Difficulty < 0 {
			time.Sleep(time.Second)
			continue
		}

		block := s.mineBlock(ctx, requirement)
		if block != nil {
			for _, stream := range s.miners {
				go func() {
					_, err := stream.AddNewBlock(ctx, block)
					if err != nil {
						// remove if error is not connected
					}
				}()
			}
			log.Printf("Successfully submitted block %d", block.Height)
			break
		}
	}
}

func (s *Miner) mineBlock(ctx context.Context, req *pb.NewBlockRequirementsResponse) *pb.Block {
	// Create a new block
	block := &pb.Block{
		Header: &pb.BlockHeader{
			Bits:          uint32(req.Difficulty),
			PrevBlockHash: req.CurrentBlockHash,
			Timestamp:     time.Now().Unix(),
		},
		Height: req.NextBlockHeight,
		Data:   fmt.Sprintf("Block %d mined by node %s:%d", req.NextBlockHeight, s.nodeInfo.Host, s.nodeInfo.Port),
	}

	// Try to find a nonce that satisfies the difficulty
	for nonce := uint64(0); nonce < 1000000000 && ctx.Err() == nil; nonce++ {
		block.Header.Nonce = nonce
		hash := utils.CalculateBlockHash(block.Header)
		if err := utils.ValidateBlock(block.GetHeader(), req.CurrentBlockHash, hash); err == nil {
			block.Hash = hash
			return block
		}
	}

	return nil
}

func (s *Miner) stopMining() {

}
