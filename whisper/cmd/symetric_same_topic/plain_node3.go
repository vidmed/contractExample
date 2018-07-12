package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

func main()  {
	IDFile := "./plain_nodeID3"
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

	// enode of boot mode
	//cfg.ArgEnode = "enode://bab2d451dcead0ac4eadfda3be9f86eb62e8c2a2158d5a88689759c6d0a4152b3f5fa970095fc7c2a7655435ba9b5d9fdadd000f37f9db47e672973a27da9408@127.0.0.1:30348"
	cfg.ArgEnode = "enode://8d4195e57458c88357272b2f7936f46739244c82288cfc5277f3ccdc8d0e52fcdf6796d461d2c1a4bbecadadc8855e1cb0a0abee0644bb89cddfde6622ddc97c@127.0.0.1:30349"
	cfg.ArgIDFile = IDFile

	cfg.ArgTopic ="746f7031"
	cfg.ArgSymPass = "321"

	wnode.StartNode(cfg)
}

