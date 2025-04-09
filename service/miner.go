package service

import (
	pb "blockchain/proto"
	"blockchain/utils"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Miner struct {
	master        pb.MasterClient
	blockServer   *BlockchainServer
	mMiners       map[string]pb.BlockchainClient
	mGrpcConn     map[string]*grpc.ClientConn
	nodeInfo      pb.NodeInfo
	mu            *sync.RWMutex
	cancelMinning context.CancelFunc
	random        rand.Source64
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
		mMiners: map[string]pb.BlockchainClient{
			masterKey: pb.NewBlockchainClient(conn),
		},
		mGrpcConn: make(map[string]*grpc.ClientConn),
		nodeInfo:  pb.NodeInfo{Host: host, Port: port, Status: "ACTIVE"},
		mu:        &sync.RWMutex{},
		random:    rand.New(rand.NewSource(time.Now().UnixMicro())),
	}, nil
}

// func connect[T pb.BlockchainClient](ctx context.Context, m *Miner, nodeInfo *pb.NodeInfo, builder func(cc grpc.ClientConnInterface) T) (T, error) {
// 	key := fmt.Sprintf("%s:%d", nodeInfo.GetHost(), nodeInfo.GetPort())
// 	conn, err := grpc.NewClient(key, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		var zero T // Zero value of T, since we can't return nil directly for a generic type
// 		return zero, fmt.Errorf("failed to connect to master: %w", err)
// 	}
// 	defer conn.Close() // Ensure the connection is closed after use

// 	client := builder(conn)

// 	m.mu.Lock()
// 	m.mMiners[key] = client
// 	m.mGrpcConn[key] = conn
// 	m.mu.Unlock()

// 	return client, nil
// }

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
			fmt.Printf("New event from %s:%d\n", nodeInfo.GetHost(), nodeInfo.GetPort())
			if err != nil {
				continue
			}
			if nodeInfo.Host != "" && nodeInfo.Port > 0 {
				minerKey := fmt.Sprintf("%s:%d", nodeInfo.Host, nodeInfo.Port)
				if nodeInfo.GetStatus() == "ACTIVE" {
					conn, err := grpc.NewClient(minerKey, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						fmt.Println("failed to connect to " + minerKey)
						continue
					}
					fmt.Println("Add miner ", minerKey)
					s.mu.Lock()
					s.mMiners[minerKey] = pb.NewBlockchainClient(conn)
					s.mGrpcConn[minerKey] = conn
					s.mu.Unlock()
				} else {
					fmt.Println("Remove miner ", minerKey)
					s.mu.Lock()
					conn := s.mGrpcConn[minerKey]
					delete(s.mGrpcConn, minerKey)
					delete(s.mMiners, minerKey)
					s.mu.Unlock()
					if conn != nil {
						conn.Close()
					}
				}
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
			stream pb.Master_RegisterNewBlockHeaderClient
			header pb.BlockHeader
			err    error
		)
		if err := backoff.Retry(func() error {
			stream, err = s.master.RegisterNewBlockHeader(ctx, &s.nodeInfo)
			return err
		}, backoff.NewExponentialBackOff()); err != nil {
			chErr <- err
			return
		}
		for err == nil {
			err = stream.RecvMsg(&header)
			fmt.Printf("new requirement from master: NextHeigth: %d, Difficulty: %d\n", header.Height, header.Bits)
			if err != nil {
				continue
			}
			s.stopMining()
			if header.Bits > 0 {
				ctx, cancel := context.WithCancel(ctx)
				s.cancelMinning = cancel
				s.blockServer.SetNewBlockHeader(
					header.GetHeight(),
					header.GetPrevBlockHash(),
					header.GetBits())
				go s.startMining(ctx, &header)
			}
		}
		chErr <- nil
	}()
	return chErr
}

func (s *Miner) OnNewBlock(ctx context.Context, blk *pb.Block) error {
	_, err := s.blockServer.AddNewBlock(ctx, blk)
	if err != nil {
		return err
	}
	if s.cancelMinning != nil {
		s.cancelMinning()
	}
	return nil
}

func (s *Miner) startMining(ctx context.Context, requirement *pb.BlockHeader) {
	for {
		if requirement == nil || requirement.Bits == 0 {
			break
		}

		block := s.mineBlock(ctx, requirement)
		if block != nil {
			for _, stream := range s.mMiners {
				go func() {
					_, err := stream.AddNewBlock(ctx, block)
					if err != nil {
						// remove if error is not connected
					}
				}()
			}
			log.Printf("Successfully submitted block %d", block.GetHeader().GetHeight())
			break
		}
	}
}

func (s *Miner) mineBlock(ctx context.Context, header *pb.BlockHeader) *pb.Block {
	// Create a new block
	block := &pb.Block{
		Header: &pb.BlockHeader{
			Bits:          header.GetBits(),
			PrevBlockHash: header.GetPrevBlockHash(),
			Height:        header.GetHeight(),
			Timestamp:     time.Now().Unix(),
		},
		Data: "Tra pham",
	}
	// Try to find a nonce that satisfies the difficulty
	for nonce := uint64(0); ctx.Err() == nil; nonce = s.random.Uint64() {
		// if nonce%1000000 == 0 {
		// 	fmt.Printf("Finding new block: nonce=%d\n", nonce)
		// }
		block.Header.Nonce = nonce
		hash, err := utils.CalculateBlockHash(block)
		if err != nil {
			return nil
		}
		block.Hash = hash
		if err := utils.ValidateBlock(block, header); err == nil {
			return block
		}
	}

	return nil
}

func (s *Miner) stopMining() {

}
