package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"blockchain/app"
	pb "blockchain/proto"

	"google.golang.org/grpc"
)

func main() {
	// Parse command line flags
	port := flag.Int("p", 50051, "Port to listen on")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	masterServer := app.NewMasterServer()
	pb.RegisterMasterServer(s, masterServer)
	blockchainServer := app.NewBlockNode(nil, masterServer.ProcessAfterNewBlock)
	pb.RegisterBlockchainServer(s, blockchainServer)

	masterServer.SetBlockchainService(blockchainServer)

	log.Printf("Master server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
