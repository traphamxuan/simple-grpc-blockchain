package service

import (
	pb "blockchain/proto"
	"blockchain/utils"
	"context"
	"encoding/hex"
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
	master         pb.MasterClient
	blockServer    *BlockchainServer
	mMiners        map[string]pb.BlockchainClient
	mGrpcConn      map[string]*grpc.ClientConn
	nodeInfo       pb.NodeInfo
	mu             *sync.RWMutex
	cancelMu       *sync.Mutex
	mCancelMinning map[uint64]context.CancelFunc
	effort         int
	signature      string
}

func NewMiner(masterHost string, masterPort int32, host string, port int32, eff int, signature string) (*Miner, error) {
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
		mGrpcConn: map[string]*grpc.ClientConn{
			masterKey: conn,
		},
		mCancelMinning: make(map[uint64]context.CancelFunc),
		nodeInfo:       pb.NodeInfo{Host: host, Port: port, Status: "ACTIVE"},
		mu:             &sync.RWMutex{},
		cancelMu:       &sync.Mutex{},
		effort:         eff,
		signature:      signature,
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

		s.blockServer = NewBlockNode(nil)
		pb.RegisterBlockchainServer(server, s.blockServer)

		log.Printf("Miner server listening at %v", lis.Addr())
		chErr <- server.Serve(lis)
	}()

	select {
	case err = <-chErr:
		return err
	case err = <-s.listenOnNewMiner(ctx):
		return err
	case err = <-s.listenOnNewBlockHeader(ctx):
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
			fmt.Printf("New miner from %s:%d:%s\n", nodeInfo.GetHost(), nodeInfo.GetPort(), nodeInfo.GetStatus())
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
					s.mu.Lock()
					s.mMiners[minerKey] = pb.NewBlockchainClient(conn)
					s.mGrpcConn[minerKey] = conn
					s.mu.Unlock()
					fmt.Println("Add miner ", minerKey)
				} else {
					s.mu.Lock()
					conn := s.mGrpcConn[minerKey]
					delete(s.mGrpcConn, minerKey)
					delete(s.mMiners, minerKey)
					s.mu.Unlock()
					fmt.Println("Remove miner ", minerKey)
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

func (s *Miner) listenOnNewBlockHeader(ctx context.Context) chan error {
	chErr := make(chan error)
	go func() {
		var (
			stream         pb.Master_RegisterNewBlockHeaderClient
			header         pb.BlockHeader
			err            error
			previousHeight uint64
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
			fmt.Printf("new Header from master: NextHeigth: %d, Difficulty: %d, hash: %s\n", header.Height, header.Bits, hex.EncodeToString(header.GetPrevBlockHash()))
			if err != nil {
				continue
			}
			if header.Bits > 0 {
				head := s.blockServer.SetNewBlockHeader(
					header.GetHeight(),
					header.GetPrevBlockHash(),
					header.GetBits())
				previousHeight = head.GetHeight()
				ctx, cancel := context.WithCancel(ctx)
				s.mu.Lock()
				s.mCancelMinning[previousHeight] = cancel
				s.mu.Unlock()
				go s.startMining(ctx, head)
			}
		}
		chErr <- nil
	}()
	return chErr
}

func (s *Miner) startMining(ctx context.Context, header *pb.BlockHeader) {
	defer fmt.Println("End of mining", header.GetHeight())
	var (
		startAt time.Time = time.Now()
		counter int64
	)
	for ctx.Err() == nil && s.blockServer.GetNewBlockHeader().GetHeight() == header.GetHeight() {
		if header == nil || header.Bits == 0 {
			break
		}

		block := s.mineBlock(header)
		if block != nil {
			block, err := s.blockServer.addNewBlock(block)
			if err != nil {
				continue
			}
			ctx = context.WithoutCancel(ctx)
			for key, stream := range s.mMiners {
				fmt.Printf("Try to send new block to %s\n", key)
				go func(ctx context.Context, key string, stream pb.BlockchainClient) {
					_, err := stream.AddNewBlock(ctx, block)
					if err != nil {
						// remove if error is not connected
						fmt.Printf("failed to send to %s: %v\n", key, err)
						return
					}
				}(ctx, key, stream)
			}
			log.Printf("Submitting block %d", block.GetHeader().GetHeight())
		}
		if counter > 1000000 {
			now := time.Now()
			duration := now.Sub(startAt)
			idle := duration * time.Duration(100-s.effort) / time.Duration(s.effort)
			// fmt.Printf("Height: %d, nonce: %d, hash: %s\n", header.GetHeight(), nonce, hex.EncodeToString(hash))
			time.Sleep(idle)
			startAt = now
			counter = 0
		}
	}
}

func (s *Miner) mineBlock(header *pb.BlockHeader) *pb.Block {
	// Create a new block
	block := &pb.Block{
		Header: &pb.BlockHeader{
			Bits:          header.GetBits(),
			PrevBlockHash: header.GetPrevBlockHash(),
			Height:        header.GetHeight(),
			Timestamp:     time.Now().Unix(),
		},
		Data: s.signature,
	}

	nonce, _ := utils.RandomUint64()
	block.Header.Nonce = nonce
	hash, err := utils.CalculateBlockHash(block)
	if err != nil {
		return nil
	}
	block.Hash = hash

	return block
}
