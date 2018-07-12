package wnode

import (
	"bytes"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/vidmed/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/whisper/whisperv6"
)

type Config struct {
	BootstrapMode  bool // boostrap node: don't initiate connection to peers, just wait for incoming connections
	ForwarderMode  bool // forwarder mode: only forward messages, neither encrypt nor decrypt messages
	MailServerMode bool // mail server mode: delivers expired messages on demand
	RequestMail    bool // request expired messages from the bootstrap server
	AsymmetricMode bool // use asymmetric encryption
	GenerateKey    bool // generate and show the private key
	FileExMode     bool // file exchange mode
	FileReader     bool // load and decrypt messages saved as files, display as plain text
	TestMode       bool // use of predefined parameters for diagnostics (password, etc.)
	EchoMode       bool // echo mode: prints some arguments for diagnostics

	ArgVerbosity int     // log verbosity level
	ArgTTL       uint    // time-to-live for messages in seconds
	ArgWorkTime  uint    // work time in seconds
	ArgMaxSize   uint    // max size of message
	ArgPoW       float64 // PoW for normal messages in float format (e.g. 2.7)
	ArgServerPoW float64 // PoW requirement for Mail Server request

	ArgIP      string // IP address and port of this node (e.g. 127.0.0.1:30303)
	ArgPub     string // public key for asymmetric encryption
	ArgDBPath  string // path to the server's DB directory
	ArgIDFile  string // file name with node id (private key)
	ArgEnode   string // bootstrap node you want to connect to (e.g. enode://e454......08d50@52.176.211.200:16428)
	ArgTopic   string // topic in hexadecimal format (e.g. 70a4beef)
	ArgSaveDir string // directory where all incoming messages will be saved as files

	// My params
	ArgSymPass string // password for symmetric encryption
	ArgPrivateKeyFile  string // file name with private key for async encrypting
	UseSelfPubKey  bool // whether to use self public Key
}

var DefaultConfig = Config{
	ArgVerbosity: int(log.LvlError),
	ArgTTL:       30,
	ArgWorkTime:  5,
	ArgMaxSize:   uint(whisperv6.DefaultMaxMessageSize),
	ArgPoW:       whisperv6.DefaultMinimumPoW,
	ArgServerPoW: whisperv6.DefaultMinimumPoW,
}

func Dump(cfg *Config) {
	var buffer bytes.Buffer
	e := toml.NewEncoder(&buffer)
	err := e.Encode(cfg)
	if err != nil {
		logger.Get().Fatal(err)
	}

	fmt.Println(
		time.Now().UTC(),
		"\n---------------------Sevice started with config:\n",
		buffer.String(),
		"\n---------------------")
}
