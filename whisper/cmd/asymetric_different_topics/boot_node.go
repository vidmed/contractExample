package main

import (
	"solidity/native_example/whisper/wnode"
	"os"
	"log"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
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

	cfg.BootstrapMode = true

	cfg.AsymmetricMode = true
	cfg.ArgPrivateKeyFile = PirvKeyFile
	cfg.UseSelfPubKey = true

	//cfg.ArgDBPath = "./db"
	cfg.ArgTopic = "746f7031"
	cfg.ArgIP = "127.0.0.1:30348"
	cfg.ArgIDFile = IDFile

	wnode.StartNode(cfg)
}
