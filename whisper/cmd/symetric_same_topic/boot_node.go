package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"log"
	"github.com/ethereum/go-ethereum/crypto"
)

func main()  {
	IDFile := "./boot_nodeID"
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

	cfg.BootstrapMode  = true
	//cfg.AsymmetricMode = true
	cfg.ArgSymPass = "123"

	cfg.ArgDBPath = "./db"
	cfg.ArgTopic ="746f7031"
	cfg.ArgIP = "127.0.0.1:30348"
	cfg.ArgIDFile = IDFile

	wnode.StartNode(cfg)
}

