// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// This is a simple Whisper node. It could be used as a stand-alone bootstrap node.
// Also, could be used for different test and diagnostics purposes.

package wnode

import (
	"bufio"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"golang.org/x/crypto/pbkdf2"
)

const quitCommand = "~Q"
const entropySize = 32

// config
var config *Config

// singletons
var (
	server     *p2p.Server
	shh        *whisper.Whisper
	done       chan struct{}
	mailServer mailserver.WMailServer
	entropy    [entropySize]byte

	input = bufio.NewReader(os.Stdin)
)

// encryption
var (
	symKey  []byte
	pub     *ecdsa.PublicKey
	asymKey *ecdsa.PrivateKey
	nodeid  *ecdsa.PrivateKey
	topic   whisper.TopicType

	asymKeyID    string
	asymFilterID string
	symFilterID  string
	symPass      string
	msPassword   string
)

func StartNode(cfg *Config) {
	config = cfg
	processArgs()
	initialize()
	run()
	shutdown()
}

func processArgs() {
	flag.Parse()

	if len(config.ArgIDFile) > 0 {
		var err error
		nodeid, err = crypto.LoadECDSA(config.ArgIDFile)
		if err != nil {
			utils.Fatalf("Failed to load file [%s]: %s.", config.ArgIDFile, err)
		}
	}

	if len(config.ArgPrivateKeyFile) > 0 {
		var err error
		asymKey, err = crypto.LoadECDSA(config.ArgPrivateKeyFile)
		if err != nil {
			utils.Fatalf("Failed to load file [%s]: %s.", config.ArgPrivateKeyFile, err)
		}
	}

	const enodePrefix = "enode://"
	if len(config.ArgEnode) > 0 {
		if (config.ArgEnode)[:len(enodePrefix)] != enodePrefix {
			config.ArgEnode = enodePrefix + config.ArgEnode
		}
	}

	if len(config.ArgTopic) > 0 {
		x, err := hex.DecodeString(config.ArgTopic)
		if err != nil {
			utils.Fatalf("Failed to parse the topic: %s", err)
		}
		topic = whisper.BytesToTopic(x)
	}

	if config.AsymmetricMode && len(config.ArgPub) > 0 {
		pub = crypto.ToECDSAPub(common.FromHex(config.ArgPub))
		if !isKeyValid(pub) {
			utils.Fatalf("invalid public key")
		}
	}

	if len(config.ArgSaveDir) > 0 {
		if _, err := os.Stat(config.ArgSaveDir); os.IsNotExist(err) {
			utils.Fatalf("Download directory '%s' does not exist", config.ArgSaveDir)
		}
	} else if config.FileExMode {
		utils.Fatalf("Parameter 'savedir' is mandatory for file exchange mode")
	}

	if config.EchoMode {
		echo()
	}
}

func echo() {
	fmt.Printf("ttl = %d \n", config.ArgTTL)
	fmt.Printf("workTime = %d \n", config.ArgWorkTime)
	fmt.Printf("pow = %f \n", config.ArgPoW)
	fmt.Printf("mspow = %f \n", config.ArgServerPoW)
	fmt.Printf("ip = %s \n", config.ArgIP)
	fmt.Printf("pub = %s \n", common.ToHex(crypto.FromECDSAPub(pub)))
	fmt.Printf("idfile = %s \n", config.ArgIDFile)
	fmt.Printf("dbpath = %s \n", config.ArgDBPath)
	fmt.Printf("boot = %s \n", config.ArgEnode)
}

func initialize() {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(config.ArgVerbosity), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	done = make(chan struct{})
	var peers []*discover.Node
	var err error

	if config.GenerateKey {
		key, err := crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("Failed to generate private key: %s", err)
		}
		k := hex.EncodeToString(crypto.FromECDSA(key))
		fmt.Printf("Random private key: %s \n", k)
		os.Exit(0)
	}

	if config.TestMode {
		symPass = "wwww" // ascii code: 0x77777777
		msPassword = "wwww"
	}
	if config.ArgSymPass != "" {
		symPass = config.ArgSymPass
		msPassword = config.ArgSymPass
	}

	if config.BootstrapMode {
		if len(config.ArgIP) == 0 {
			config.ArgIP = *scanLineA("Please enter your IP and port (e.g. 127.0.0.1:30348): ")
		}
	} else if config.FileReader {
		config.BootstrapMode = true
	} else {
		if len(config.ArgEnode) == 0 {
			config.ArgEnode = *scanLineA("Please enter the peer's enode: ")
		}
		peer := discover.MustParseNode(config.ArgEnode)
		peers = append(peers, peer)
	}

	if config.MailServerMode {
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				utils.Fatalf("Failed to read Mail Server password: %s", err)
			}
		}
	}

	cfg := &whisper.Config{
		MaxMessageSize:     uint32(config.ArgMaxSize),
		MinimumAcceptedPOW: config.ArgPoW,
	}

	shh = whisper.New(cfg)

	if config.ArgPoW != whisper.DefaultMinimumPoW {
		err := shh.SetMinimumPoW(config.ArgPoW)
		if err != nil {
			utils.Fatalf("Failed to set PoW: %s", err)
		}
	}

	if uint32(config.ArgMaxSize) != whisper.DefaultMaxMessageSize {
		err := shh.SetMaxMessageSize(uint32(config.ArgMaxSize))
		if err != nil {
			utils.Fatalf("Failed to set max message size: %s", err)
		}
	}

	if asymKey == nil {
		asymKeyID, err = shh.NewKeyPair()
		if err != nil {
			utils.Fatalf("Failed to generate a new key pair: %s", err)
		}

		asymKey, err = shh.GetPrivateKey(asymKeyID)
		if err != nil {
			utils.Fatalf("Failed to retrieve a new key pair: %s", err)
		}
	} else {
		asymKeyID, err = shh.AddKeyPair(asymKey)
		if err != nil {
			utils.Fatalf("Failed to add a new key pair: %s", err)
		}
	}

	if nodeid == nil {
		tmpID, err := shh.NewKeyPair()
		if err != nil {
			utils.Fatalf("Failed to generate a new key pair: %s", err)
		}

		nodeid, err = shh.GetPrivateKey(tmpID)
		if err != nil {
			utils.Fatalf("Failed to retrieve a new key pair: %s", err)
		}
	}

	maxPeers := 80
	if config.BootstrapMode {
		maxPeers = 800
	}

	_, err = crand.Read(entropy[:])
	if err != nil {
		utils.Fatalf("crypto/rand failed: %s", err)
	}

	if config.MailServerMode {
		shh.RegisterServer(&mailServer)
		if err := mailServer.Init(shh, config.ArgDBPath, msPassword, config.ArgServerPoW); err != nil {
			utils.Fatalf("Failed to init MailServer: %s", err)
		}
	}

	server = &p2p.Server{
		Config: p2p.Config{
			PrivateKey:     nodeid,
			MaxPeers:       maxPeers,
			Name:           common.MakeName("wnode", "6.0"),
			Protocols:      shh.Protocols(),
			ListenAddr:     config.ArgIP,
			NAT:            nat.Any(),
			BootstrapNodes: peers,
			StaticNodes:    peers,
			TrustedNodes:   peers,
		},
	}
}

func startServer() error {
	err := server.Start()
	if err != nil {
		fmt.Printf("Failed to start Whisper peer: %s.", err)
		return err
	}

	fmt.Printf("my public key: %s \n", common.ToHex(crypto.FromECDSAPub(&asymKey.PublicKey)))
	fmt.Println(server.NodeInfo().Enode)

	if config.BootstrapMode {
		configureNode()
		fmt.Println("Bootstrap Whisper node started")
	} else {
		fmt.Println("Whisper node started")
		// first see if we can establish connection, then ask for user input
		waitForConnection(true)
		configureNode()
	}

	if config.FileExMode {
		fmt.Printf("Please type the file name to be send. To quit type: '%s'\n", quitCommand)
	} else if config.FileReader {
		fmt.Printf("Please type the file name to be decrypted. To quit type: '%s'\n", quitCommand)
	} else if !config.ForwarderMode {
		fmt.Printf("Please type the message. To quit type: '%s'\n", quitCommand)
	}
	return nil
}

func isKeyValid(k *ecdsa.PublicKey) bool {
	return k.X != nil && k.Y != nil
}

func configureNode() {
	var err error
	var p2pAccept bool

	if config.ForwarderMode {
		return
	}

	if config.AsymmetricMode {
		if len(config.ArgPub) == 0 {
			if config.UseSelfPubKey {
				fmt.Println("Used self public key for listening")
				pub = &asymKey.PublicKey
			} else {
				s := scanLine("Please enter the peer's public key: ")
				b := common.FromHex(s)
				if b == nil {
					utils.Fatalf("Error: can not convert hexadecimal string")
				}
				pub = crypto.ToECDSAPub(b)
				if !isKeyValid(pub) {
					utils.Fatalf("Error: invalid public key")
				}
			}
		}
	}

	if config.RequestMail {
		p2pAccept = true
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				utils.Fatalf("Failed to read Mail Server password: %s", err)
			}
		}
	}

	if !config.AsymmetricMode && !config.ForwarderMode {
		if len(symPass) == 0 {
			symPass, err = console.Stdin.PromptPassword("Please enter the password for symmetric encryption: ")
			if err != nil {
				utils.Fatalf("Failed to read passphrase: %v", err)
			}
		}

		symKeyID, err := shh.AddSymKeyFromPassword(symPass)
		if err != nil {
			utils.Fatalf("Failed to create symmetric key: %s", err)
		}
		symKey, err = shh.GetSymKey(symKeyID)
		if err != nil {
			utils.Fatalf("Failed to save symmetric key: %s", err)
		}
		if len(config.ArgTopic) == 0 {
			generateTopic([]byte(symPass))
		}

		fmt.Printf("Filter is configured for the topic: %x \n", topic)
	}

	if config.MailServerMode {
		if len(config.ArgDBPath) == 0 {
			config.ArgDBPath = *scanLineA("Please enter the path to DB file: ")
		}
	}

	symFilter := whisper.Filter{
		KeySym:   symKey,
		Topics:   [][]byte{topic[:]},
		AllowP2P: p2pAccept,
	}
	symFilterID, err = shh.Subscribe(&symFilter)
	if err != nil {
		utils.Fatalf("Failed to install filter: %s", err)
	}

	asymFilter := whisper.Filter{
		KeyAsym:  asymKey,
		Topics:   [][]byte{topic[:]},
		AllowP2P: p2pAccept,
	}
	asymFilterID, err = shh.Subscribe(&asymFilter)
	if err != nil {
		utils.Fatalf("Failed to install filter: %s", err)
	}
}

func generateTopic(password []byte) {
	x := pbkdf2.Key(password, password, 4096, 128, sha512.New)
	for i := 0; i < len(x); i++ {
		topic[i%whisper.TopicLength] ^= x[i]
	}
}

func waitForConnection(timeout bool) {
	var cnt int
	var connected bool
	for !connected {
		time.Sleep(time.Millisecond * 50)
		connected = server.PeerCount() > 0
		if timeout {
			cnt++
			if cnt > 1000 {
				utils.Fatalf("Timeout expired, failed to connect")
			}
		}
	}

	fmt.Println("Connected to peer.")
}

func run() {
	err := startServer()
	if err != nil {
		return
	}
	defer server.Stop()
	shh.Start(nil)
	defer shh.Stop()

	if !config.ForwarderMode {
		go messageLoop()
	}

	if config.RequestMail {
		requestExpiredMessagesLoop()
	} else if config.FileExMode {
		sendFilesLoop()
	} else if config.FileReader {
		fileReaderLoop()
	} else {
		sendLoop()
	}
}

func shutdown() {
	close(done)
	mailServer.Close()
}

func sendLoop() {
	for {
		s := scanLine("")
		if s == quitCommand {
			fmt.Println("Quit command received")
			return
		}
		sendMsg([]byte(s))
		if config.AsymmetricMode {
			// print your own message for convenience,
			// because in asymmetric mode it is impossible to decrypt it
			timestamp := time.Now().Unix()
			from := crypto.PubkeyToAddress(asymKey.PublicKey)
			fmt.Printf("\n%d <%x>: %s\n", timestamp, from, s)
		}
	}
}

func sendFilesLoop() {
	for {
		s := scanLine("")
		if s == quitCommand {
			fmt.Println("Quit command received")
			return
		}
		b, err := ioutil.ReadFile(s)
		if err != nil {
			fmt.Printf(">>> Error: %s \n", err)
		} else {
			h := sendMsg(b)
			if (h == common.Hash{}) {
				fmt.Printf(">>> Error: message was not sent \n")
			} else {
				timestamp := time.Now().Unix()
				from := crypto.PubkeyToAddress(asymKey.PublicKey)
				fmt.Printf("\n%d <%x>: sent message with hash %x\n", timestamp, from, h)
			}
		}
	}
}

func fileReaderLoop() {
	watcher1 := shh.GetFilter(symFilterID)
	watcher2 := shh.GetFilter(asymFilterID)
	if watcher1 == nil && watcher2 == nil {
		fmt.Println("Error: neither symmetric nor asymmetric filter is installed")
		return
	}

	for {
		s := scanLine("")
		if s == quitCommand {
			fmt.Println("Quit command received")
			return
		}
		raw, err := ioutil.ReadFile(s)
		if err != nil {
			fmt.Printf(">>> Error: %s \n", err)
		} else {
			env := whisper.Envelope{Data: raw} // the topic is zero
			msg := env.Open(watcher1)          // force-open envelope regardless of the topic
			if msg == nil {
				msg = env.Open(watcher2)
			}
			if msg == nil {
				fmt.Printf(">>> Error: failed to decrypt the message \n")
			} else {
				printMessageInfo(msg)
			}
		}
	}
}

func scanLine(prompt string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt)
	}
	txt, err := input.ReadString('\n')
	if err != nil {
		utils.Fatalf("input error: %s", err)
	}
	txt = strings.TrimRight(txt, "\n\r")
	return txt
}

func scanLineA(prompt string) *string {
	s := scanLine(prompt)
	return &s
}

func scanUint(prompt string) uint32 {
	s := scanLine(prompt)
	i, err := strconv.Atoi(s)
	if err != nil {
		utils.Fatalf("Fail to parse the lower time limit: %s", err)
	}
	return uint32(i)
}

func sendMsg(payload []byte) common.Hash {
	params := whisper.MessageParams{
		Src:      asymKey,
		Dst:      pub,
		KeySym:   symKey,
		Payload:  payload,
		Topic:    topic,
		TTL:      uint32(config.ArgTTL),
		PoW:      config.ArgPoW,
		WorkTime: uint32(config.ArgWorkTime),
	}

	msg, err := whisper.NewSentMessage(&params)
	if err != nil {
		utils.Fatalf("failed to create new message: %s", err)
	}

	envelope, err := msg.Wrap(&params)
	if err != nil {
		fmt.Printf("failed to seal message: %v \n", err)
		return common.Hash{}
	}

	err = shh.Send(envelope)
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
		return common.Hash{}
	}

	return envelope.Hash()
}

func messageLoop() {
	sf := shh.GetFilter(symFilterID)
	if sf == nil {
		utils.Fatalf("symmetric filter is not installed")
	}

	af := shh.GetFilter(asymFilterID)
	if af == nil {
		utils.Fatalf("asymmetric filter is not installed")
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	for {
		select {
		case <-ticker.C:
			m1 := sf.Retrieve()
			m2 := af.Retrieve()
			messages := append(m1, m2...)
			for _, msg := range messages {
				reportedOnce := false
				if !config.FileExMode && len(msg.Payload) <= 2048 {
					printMessageInfo(msg)
					reportedOnce = true
				}

				// All messages are saved upon specifying argSaveDir.
				// fileExMode only specifies how messages are displayed on the console after they are saved.
				// if fileExMode == true, only the hashes are displayed, since messages might be too big.
				if len(config.ArgSaveDir) > 0 {
					writeMessageToFile(config.ArgSaveDir, msg, !reportedOnce)
				}
			}
		case <-done:
			return
		}
	}
}

func printMessageInfo(msg *whisper.ReceivedMessage) {
	timestamp := fmt.Sprintf("%d", msg.Sent) // unix timestamp for diagnostics
	text := string(msg.Payload)

	var address common.Address
	if msg.Src != nil {
		address = crypto.PubkeyToAddress(*msg.Src)
	}

	if whisper.IsPubKeyEqual(msg.Src, &asymKey.PublicKey) {
		fmt.Printf("\nReal message %s <%x>: %s\n", timestamp, address, text) // message from myself
	} else {
		fmt.Printf("\nReal message %s [%x]: %s\n", timestamp, address, text) // message from a peer
	}
}

func writeMessageToFile(dir string, msg *whisper.ReceivedMessage, show bool) {
	if len(dir) == 0 {
		return
	}

	timestamp := fmt.Sprintf("%d", msg.Sent)
	name := fmt.Sprintf("%x", msg.EnvelopeHash)

	var address common.Address
	if msg.Src != nil {
		address = crypto.PubkeyToAddress(*msg.Src)
	}

	env := shh.GetEnvelope(msg.EnvelopeHash)
	if env == nil {
		fmt.Printf("\nUnexpected error: envelope not found: %x\n", msg.EnvelopeHash)
		return
	}

	// this is a sample code; uncomment if you don't want to save your own messages.
	//if whisper.IsPubKeyEqual(msg.Src, &asymKey.PublicKey) {
	//	fmt.Printf("\n%s <%x>: message from myself received, not saved: '%s'\n", timestamp, address, name)
	//	return
	//}

	fullpath := filepath.Join(dir, name)
	err := ioutil.WriteFile(fullpath, env.Data, 0644)
	if err != nil {
		fmt.Printf("\n%s {%x}: message received but not saved: %s\n", timestamp, address, err)
	} else if show {
		fmt.Printf("\n%s {%x}: message received and saved as '%s' (%d bytes)\n", timestamp, address, name, len(env.Data))
	}
}

func requestExpiredMessagesLoop() {
	var key, peerID, bloom []byte
	var timeLow, timeUpp uint32
	var t string
	var xt whisper.TopicType

	keyID, err := shh.AddSymKeyFromPassword(msPassword)
	if err != nil {
		utils.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}
	key, err = shh.GetSymKey(keyID)
	if err != nil {
		utils.Fatalf("Failed to save symmetric key for mail request: %s", err)
	}
	peerID = extractIDFromEnode(config.ArgEnode)
	shh.AllowP2PMessagesFromPeer(peerID)

	for {
		timeLow = scanUint("Please enter the lower limit of the time range (unix timestamp): ")
		timeUpp = scanUint("Please enter the upper limit of the time range (unix timestamp): ")
		t = scanLine("Enter the topic (hex). Press enter to request all messages, regardless of the topic: ")
		if len(t) == whisper.TopicLength*2 {
			x, err := hex.DecodeString(t)
			if err != nil {
				fmt.Printf("Failed to parse the topic: %s \n", err)
				continue
			}
			xt = whisper.BytesToTopic(x)
			bloom = whisper.TopicToBloom(xt)
			obfuscateBloom(bloom)
		} else if len(t) == 0 {
			bloom = whisper.MakeFullNodeBloom()
		} else {
			fmt.Println("Error: topic is invalid, request aborted")
			continue
		}

		if timeUpp == 0 {
			timeUpp = 0xFFFFFFFF
		}

		data := make([]byte, 8, 8+whisper.BloomFilterSize)
		binary.BigEndian.PutUint32(data, timeLow)
		binary.BigEndian.PutUint32(data[4:], timeUpp)
		data = append(data, bloom...)

		var params whisper.MessageParams
		params.PoW = config.ArgServerPoW
		params.Payload = data
		params.KeySym = key
		params.Src = asymKey
		params.WorkTime = 5

		msg, err := whisper.NewSentMessage(&params)
		if err != nil {
			utils.Fatalf("failed to create new message: %s", err)
		}
		env, err := msg.Wrap(&params)
		if err != nil {
			utils.Fatalf("Wrap failed: %s", err)
		}

		err = shh.RequestHistoricMessages(peerID, env)
		if err != nil {
			utils.Fatalf("Failed to send P2P message: %s", err)
		}

		time.Sleep(time.Second * 5)
	}
}

func extractIDFromEnode(s string) []byte {
	n, err := discover.ParseNode(s)
	if err != nil {
		utils.Fatalf("Failed to parse enode: %s", err)
	}
	return n.ID[:]
}

// obfuscateBloom adds 16 random bits to the the bloom
// filter, in order to obfuscate the containing topics.
// it does so deterministically within every session.
// despite additional bits, it will match on average
// 32000 times less messages than full node's bloom filter.
func obfuscateBloom(bloom []byte) {
	const half = entropySize / 2
	for i := 0; i < half; i++ {
		x := int(entropy[i])
		if entropy[half+i] < 128 {
			x += 256
		}

		bloom[x/8] = 1 << uint(x%8) // set the bit number X
	}
}
