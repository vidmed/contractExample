package main

import (
	"log"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"os"
	"github.com/davecgh/go-spew/spew"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"io/ioutil"
)

func main(){
	cfg := new(runtime.Config)
	setDefaults(cfg)

	pathAbi := "/home/dmedvedev/go/src/solidity/native_example/contract/greeter.abi"
	pathBin := "/home/dmedvedev/go/src/solidity/native_example/contract/greeter.bin"
	method := "greet"
	binHex, err := ioutil.ReadFile(pathBin)
	if err != nil {
		log.Fatal(err)
	}
	bin := common.Hex2Bytes(string(binHex))
	fmt.Println("Contract BIN (compiled)")
	spew.Dump(bin)


	out, contractAddress, _, err := runtime.Create(bin, cfg)
	fmt.Printf("contractAddress - %s\n", contractAddress.Hex())
	fmt.Println("Create contract result")
	spew.Dump(out)

	//---------------------------------
	//contractAddress := common.HexToAddress("0x491d1B132e6E790C655D9a6717e42Ef419fe6147")
	//-------------------------------

	f, err := os.Open(pathAbi)
	if err != nil {
		log.Fatal("os.Open(pathAbi)", err)
	}
	ABI, err := abi.JSON(f)
	if err != nil {
		log.Fatal("abi.JSON(f)", err)
	}
	input, err := ABI.Pack(method)
	if err != nil {
		log.Fatal("a.Pack()", err)
	}

	contractResult, _, err := runtime.Call(contractAddress, input, cfg)
	if err != nil {
		log.Fatal("runtime.Call - ", err)
	}
	//res := (hexutil.Bytes)(contractResult)
	fmt.Println("Call contract method result")
	spew.Dump(contractResult)

	var res  = new(string)

	// ------------------
	err = ABI.Unpack(res, method, contractResult)
	if err != nil {
		log.Fatal("meth.Outputs.Unpack - ", err)
	}
	spew.Dump(res)
	// --------------------------


	//meth, ok := ABI.Methods[method]
	//if !ok {
	//	log.Fatal("a.Methods[method] - not found")
	//}
	//err = meth.Outputs.Unpack(res, contractResult)
	//if err != nil {
	//	log.Fatal("meth.Outputs.Unpack - ", err)
	//}
	//spew.Dump(res)
}


// sets defaults on the config
func setDefaults(cfg *runtime.Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainId:        big.NewInt(1),
			HomesteadBlock: new(big.Int),
			DAOForkBlock:   new(big.Int),
			DAOForkSupport: false,
			EIP150Block:    new(big.Int),
			EIP155Block:    new(big.Int),
			EIP158Block:    new(big.Int),
		}
	}
	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int).SetInt64(1)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int).SetInt64(102509)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
	cfg.Coinbase =common.HexToAddress("0xCeEBbc570B37757903278A7659f441198264d395")
	cfg.Origin =common.HexToAddress("0xCeEBbc570B37757903278A7659f441198264d395")

	db, err := ethdb.NewLDBDatabase("/home/dmedvedev/go/src/node-docker-deploy/files/chain_bch02_1/geth/chaindata", 0, 0)
	if err != nil {
		log.Fatal("ethdb.NewLDBDatabase - ", err)
	}

	cfg.State, err = state.New(common.HexToHash("0x0539403de037deb0a6385e1c82613dd63588bba386047df7041d4cfb56233112"), state.NewDatabase(db))
	if err != nil {
		log.Fatal("state.New - ", err)
	}
}