package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

func main()  {
	IDFile := "./plain_nodeID1"
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

	PirvKeyFile := "./private_key"
	if _, err := os.Stat(PirvKeyFile); os.IsNotExist(err) { // create private key
		//generate
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			log.Fatalf("Failed to generate privateKey: %s.\n", err)
		}
		//save
		err = crypto.SaveECDSA(PirvKeyFile, privateKey)
		if err != nil {
			log.Fatalf("Failed to save privateKey file [%s]: %s.\n", PirvKeyFile, err)
		}
		log.Printf("private Key file [%s] saved.", PirvKeyFile)
	}


	cfg := &wnode.DefaultConfig

	cfg.AsymmetricMode = true
	cfg.ArgPrivateKeyFile = PirvKeyFile
	cfg.UseSelfPubKey = true

	// enode of boot mode
	cfg.ArgEnode = "enode://bab2d451dcead0ac4eadfda3be9f86eb62e8c2a2158d5a88689759c6d0a4152b3f5fa970095fc7c2a7655435ba9b5d9fdadd000f37f9db47e672973a27da9408@127.0.0.1:30348"
	cfg.ArgIDFile = IDFile

	cfg.ArgTopic ="746f7032"

	wnode.StartNode(cfg)
}

