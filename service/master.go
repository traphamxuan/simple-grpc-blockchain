package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "blockchain/proto"

	"google.golang.org/grpc"
)

type MasterServer struct {
	blockchainServer *BlockchainServer

	pb.UnimplementedMasterServer
	mu                  *sync.RWMutex
	mNodeInfo           map[string]*pb.NodeInfo
	mNodeStream         map[string]pb.Master_RegisterNodeServer
	mNodeNewBlockStream map[string]pb.Master_RegisterNewBlockRequirementsServer
}

func NewMasterServer() *MasterServer {
	return &MasterServer{
		mu:                  &sync.RWMutex{},
		mNodeInfo:           make(map[string]*pb.NodeInfo),
		mNodeStream:         make(map[string]pb.Master_RegisterNodeServer),
		mNodeNewBlockStream: make(map[string]pb.Master_RegisterNewBlockRequirementsServer),
	}
}

func (s *MasterServer) Start(ctx context.Context, port int32) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	masterServer := NewMasterServer()
	pb.RegisterMasterServer(server, masterServer)
	blockchainServer := NewBlockNode(nil, masterServer.ProcessAfterNewBlock)
	pb.RegisterBlockchainServer(server, blockchainServer)

	masterServer.SetBlockchainService(blockchainServer)

	log.Printf("Master server listening at %v", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	return nil
}

func (s *MasterServer) SetBlockchainService(service *BlockchainServer) {
	s.blockchainServer = service
}

func (s *MasterServer) RegisterNode(req *pb.NodeInfo, stream pb.Master_RegisterNodeServer) error {
	nodeKey := fmt.Sprintf("%s:%d", req.Host, req.Port)
	fmt.Println("New node at ", nodeKey)
	s.mu.Lock()

	s.mNodeInfo[nodeKey] = req
	s.mNodeStream[nodeKey] = stream
	s.mu.Unlock()

	// Send list of nodes
	for key, nodeInfo := range s.mNodeInfo {
		if key == nodeKey {
			continue
		}
		if err := stream.Send(nodeInfo); err != nil {
			goto END_OF_NODE_LIFE
		}
	}

	// Notify all others node about the new one
	for key, nodeStream := range s.mNodeStream {
		if key == nodeKey {
			continue
		}
		nodeStream.Send(req)
	}

	// Keep the stream open and wait for context cancellation
	<-stream.Context().Done()

END_OF_NODE_LIFE:
	// Clean up when stream is closed
	fmt.Println("Remove node at ", nodeKey)
	s.mu.Lock()
	delete(s.mNodeInfo, nodeKey)
	delete(s.mNodeStream, nodeKey)
	s.mu.Unlock()

	return nil
}

func (s *MasterServer) RegisterNewBlockRequirements(req *pb.NodeInfo, stream pb.Master_RegisterNewBlockRequirementsServer) error {
	nodeKey := fmt.Sprintf("%s:%d", req.Host, req.Port)
	fmt.Println("New block node at ", nodeKey)

	s.mu.Lock()
	s.mNodeInfo[nodeKey] = req
	s.mNodeNewBlockStream[nodeKey] = stream
	s.mu.Unlock()

	// Keep the stream open and wait for context cancellation
	<-stream.Context().Done()

	// Clean up when stream is closed
	fmt.Println("Remove block node at ", nodeKey)
	s.mu.Lock()
	delete(s.mNodeInfo, nodeKey)
	delete(s.mNodeStream, nodeKey)
	s.mu.Unlock()

	return nil
}

func (s *MasterServer) notifyNodesOfNewRequirements(requirement *pb.NewBlockRequirementsResponse) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var wg sync.WaitGroup

	for key, stream := range s.mNodeNewBlockStream {
		wg.Add(1)
		go func(key string, stream pb.Master_RegisterNewBlockRequirementsServer) {
			defer wg.Done()
			if err := stream.Send(requirement); err != nil {
				log.Printf("Failed to notify node %s of new requirements: %v", key, err)
			}
		}(key, stream)
	}
	wg.Wait()
}

func (s *MasterServer) ProcessAfterNewBlock(ctx context.Context, block *pb.Block) error {
	requirement := s.blockchainServer.SetRequirement(ctx,
		block.GetHeight()+1,
		block.GetHash(),
		0x1F,
	)
	go s.notifyNodesOfNewRequirements(requirement)
	return nil
}
