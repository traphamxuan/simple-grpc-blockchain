package main

import (
	"blockchain/app"
	"context"
	"flag"
)

// func (s *Miner) listenOnMaster(host string, port int32) {
// 	stream, err := s.master.RegisterNode(context.Background(), &pb.NodeRequest{
// 		Host: host,
// 		Port: port,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	var msg pb.FarmingRequirementsResponse
// 	for err == nil {
// 		err = stream.RecvMsg(&msg)
// 		if err != nil {
// 			err = backoff.Retry(func() error {
// 				stream, err = s.master.RegisterNode(context.Background(), &pb.NodeRequest{
// 					Host: host,
// 					Port: port,
// 				})
// 				return err
// 			}, backoff.NewExponentialBackOff())
// 			if err != nil {
// 				panic(err)
// 			}
// 			continue
// 		}
// 		fmt.Printf("Got requirement for new block from master \nblockHash: %s\ndifficulty: %d\nNext Height: %d \n",
// 			msg.CurrentBlockHash, msg.Difficulty, msg.NextBlockHeight)
// 		// if farming, stop farm
// 		// start farming here
// 	}
// }

func main() {
	ctx := context.Background()
	// These would typically come from command line arguments
	// Parse command line flags
	masterPort := flag.Int("mp", 50051, "Master port to connect")
	masterHost := flag.String("mh", "localhost", "Master host to conect")
	nodePort := flag.Int("p", 50052, "Port of node to listen")
	nodeHost := flag.String("h", "localhost", "Host address of node")

	flag.Parse()

	minerServer, err := app.NewMiner(*masterHost, int32(*masterPort), *nodeHost, int32(*nodePort))
	if err != nil {
		panic(err)
	}
	minerServer.Start(ctx)
}
