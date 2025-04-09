package main

import (
	"blockchain/service"
	"context"
	"flag"
	"fmt"
)

func main() {
	ctx := context.Background()
	ctxApp, cancelFnc := context.WithCancel(ctx)
	defer cancelFnc()
	// These would typically come from command line arguments
	// Parse command line flags
	masterPort := flag.Int("mp", 0, "Master port to connect")
	masterHost := flag.String("mh", "localhost", "Master host to conect")
	nodePort := flag.Int("p", 50052, "Port of node to listen")
	nodeHost := flag.String("h", "localhost", "Host address of node")

	flag.Parse()

	if masterPort != nil && *masterPort > 0 {
		fmt.Println("Run as miner node")
		minerServer, err := service.NewMiner(*masterHost, int32(*masterPort), *nodeHost, int32(*nodePort))
		if err != nil {
			panic(err)
		}
		minerServer.Start(ctxApp)
		return
	}

	fmt.Println("Run as master node")
	masterServer := service.NewMasterServer()
	if err := masterServer.Start(ctxApp, int32(*nodePort)); err != nil {
		panic(err)
	}
}
