package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

func main()  {
	IDFile := "./receive_nodeID1"
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
	cfg.RequestMail  = true

	// enode of boot mode
	cfg.ArgEnode = "enode://c6fefcc64c1557d4df56a2f8c0dc3ae53f0ad31c82002848e500edcf00c71426dce424078e96eaf03656b81f9b8be4c8ed1438993524294a0f1ff6aef98d1ad2@127.0.0.1:30348"
	cfg.ArgIDFile = IDFile

	cfg.ArgTopic ="746f7031"
	cfg.ArgSymPass = "123"

	wnode.StartNode(cfg)
}

