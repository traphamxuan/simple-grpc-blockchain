package main

import (
	"blockchain/service"
	"context"
	"flag"
)

func main() {
	ctx := context.Background()
	// These would typically come from command line arguments
	// Parse command line flags
	masterPort := flag.Int("mp", 0, "Master port to connect")
	masterHost := flag.String("mh", "localhost", "Master host to conect")
	nodePort := flag.Int("p", 50052, "Port of node to listen")
	nodeHost := flag.String("h", "localhost", "Host address of node")

	flag.Parse()

	if masterPort != nil && *masterPort > 0 {
		// Start as miner connect to master
		minerServer, err := service.NewMiner(*masterHost, int32(*masterPort), *nodeHost, int32(*nodePort))
		if err != nil {
			panic(err)
		}
		minerServer.Start(ctx)
		return
	}

	masterServer := service.NewMasterServer()
	if err := masterServer.Start(ctx, int32(*nodePort)); err != nil {
		panic(err)
	}
}
