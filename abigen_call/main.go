package main

import (
	"log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/davecgh/go-spew/spew"
	"solidity/native_example/contract"
	"context"
	"os"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Generate a new random account and a funded simulator
	//privateKey, _ := crypto.GenerateKey()
	//auth := bind.NewKeyedTransactor(privateKey)
	file, err := os.Open("/home/dmedvedev/go/src/node-docker-deploy/files/keystore/UTC--2017-09-15T07-13-16.119605454Z--ceebbc570b37757903278a7659f441198264d395")
	if err != nil {
		log.Fatalf("Failed to open keyin file: %v", err)
	}
	auth, err := bind.NewTransactor(file, "qwe")
	if err != nil {
		log.Fatalf("Failed to NewTransactor(): %v", err)
	}

	// backend instance
	backend, err := ethclient.Dial("/home/dmedvedev/go/src/node-docker-deploy/files/chain_bch02_1/geth.ipc")
	if err != nil {
		log.Fatalf("Failed to dialing node: %v", err)
	}
	defer backend.Close()

	//backend := backends.NewSimulatedBackend(core.GenesisAlloc{auth.From:core.GenesisAccount{Balance:big.NewInt(9999999999999)}})

	// -----------------USE THIS FOR NEW CONTRACT

	// Deploy a token contract on the simulated blockchain
	contractAddress, tx, _, err := contract.DeployContract(auth, backend)
	if err != nil {
		log.Fatalf("Failed to deploy new token contract: %v", err)
	}
	fmt.Printf("contractAddress - %s\n", contractAddress.Hex())
	fmt.Printf("tx.Hash() - %s\n", tx.Hash().Hex())
	//time.Sleep(1*time.Second)

	// -----------------USE THIS FOR EXISTING CONTRACT
	//contractAddress := common.HexToAddress("0x491d1B132e6E790C655D9a6717e42Ef419fe6147")

	contractInstance, err:= contract.NewContract(contractAddress, backend)

	res, err := contractInstance.Greet(&bind.CallOpts{Pending:true, From:auth.From, Context:context.Background()})
	if err != nil {
		log.Fatalf("Failed to call Greet() of contruct: %v", err)
	}
	spew.Dump(res)
}
