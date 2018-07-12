package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

func main()  {
	IDFile := "./plain_nodeID2"
	if _, err := os.Stat(IDFile); os.IsNotExist(err) { // create ID file
		//generate
		nodeid, err := crypto.GenerateKey()
		if err != nil {
			log.Fatalf("Failed to generate nodeID: %s.\n", err)
		}
		//save
			err = crypto.SaveECDSA(IDFile, nodeid)
			if err != nil {
				log.Fatalf("Failed to save ID file [%s]: %s.\n", IDFile, err)
			}
			log.Printf("ID file [%s] saved.", IDFile)
	}


	cfg := &wnode.DefaultConfig

	cfg.AsymmetricMode = true
	cfg.ArgPub ="0x04e2428c5a29283665b94d3c4c8c17156c79f41009619596956e1fb41517dfcfa0b72b6bf4399c31735b952d45e7635f15765a5396bab70e87f7afb88728defe9a"

	// enode of boot mode
	cfg.ArgEnode = "enode://bab2d451dcead0ac4eadfda3be9f86eb62e8c2a2158d5a88689759c6d0a4152b3f5fa970095fc7c2a7655435ba9b5d9fdadd000f37f9db47e672973a27da9408@127.0.0.1:30348"
	cfg.ArgIDFile = IDFile

	cfg.ArgTopic ="746f7031"

	wnode.StartNode(cfg)
}

