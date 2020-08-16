package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/armon/go-socks5"
	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
)

type Config struct {
	TorConfig       tor.StartConf
	TorListenConfig tor.ListenConf
	SOCKS5Config    socks5.Config
	Timeout         int
	HiddenPort      int
}

var creator = libtor.Creator

type LibTorWrapper struct{}

func (LibTorWrapper) New(ctx context.Context, args ...string) (process.Process, error) {
	return creator.New(ctx, args...)
}

func parseArgs() Config {
	config := Config{}

	config.TorConfig = tor.StartConf{}
	config.TorConfig.ProcessCreator = LibTorWrapper{}
	config.TorConfig.DebugWriter = os.Stderr
	flag.StringVar(&config.TorConfig.DataDir, "data-dir", "", "Where Tor data is stored. If not defined, a directory is created")
	var flagHiddenSrvPort int
	flag.IntVar(&flagHiddenSrvPort, "hidden-port", 80, "Port for onion service")
	flag.IntVar(&config.Timeout, "timeout", 180, "Timeout in seconds for Tor setup")
	config.TorListenConfig = tor.ListenConf{}

	var socks5User, socks5Pass string
	flag.StringVar(&socks5User, "socks5-user", "", "SOCKS5 user. Optional")
	flag.StringVar(&socks5Pass, "socks5-pass", "", "SOCKS5 pass. Optional")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()
	if config.TorConfig.DataDir == "" {
		currentdir, _ := os.Getwd()
		datadir, err := ioutil.TempDir(currentdir, "data-dir-")
		if err != nil {
			log.Panicf("Cannot create data-dir. %v", err)
		}
		config.TorConfig.DataDir = datadir
	}

	if socks5User != "" && socks5Pass != "" {
		cred := socks5.StaticCredentials{socks5User: socks5Pass}
		config.SOCKS5Config = socks5.Config{Credentials: cred}
	} else if socks5User != "" {
		// User but not Password
		fmt.Fprintf(os.Stderr, "SOCKS5 Pass parameter (-socks5-pass) is required when specifying SOCKS5 User (-sock5-user).\n\n")
		flag.Usage()
		os.Exit(1)
	} else if socks5Pass != "" {
		// Password but not User
		fmt.Fprintf(os.Stderr, "SOCKS5 User parameter (-socks5-user) is required when specifying SOCKS5 Pass (-sock5-pass).\n\n")
		flag.Usage()
		os.Exit(1)
	} else {
		// Without credentials
		config.SOCKS5Config = socks5.Config{}
	}

	config.TorListenConfig.RemotePorts = []int{flagHiddenSrvPort}
	return config
}

func main() {
	config := parseArgs()
	// Start tor with some defaults + elevated verbosity
	fmt.Println("Starting and registering onion service, please wait a bit...")
	t, err := tor.Start(nil, &config.TorConfig)
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	defer t.Close()

	// Wait at most a few minutes to publish the service
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	if _, err := os.Stat(config.TorConfig.DataDir + "/keys/onion.pem"); os.IsNotExist(err) {
		// No key, so force creation
		// openssl genrsa -out $datadir/keys/onion.pem 1024
		key, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			log.Panicf("Failed to generate RSA private key")
		}
		config.TorListenConfig.Key = key

		keyfile, err := os.Create(config.TorConfig.DataDir + "/keys/onion.pem")
		if err != nil {
			log.Panicf("Cannot save RSA private key. %v", err)
		}
		pem.Encode(keyfile, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})

	} else {
		// Found key for onion service
		buff, err := ioutil.ReadFile(config.TorConfig.DataDir + "/keys/onion.pem")
		block, _ := pem.Decode(buff)
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			log.Panicf("Wrong private key format")
		}
		config.TorListenConfig.Key = key
	}

	// Create an onion service to listen on any port but show as 80
	onion, err := t.Listen(ctx, &config.TorListenConfig)
	if err != nil {
		log.Panicf("Failed to create onion service: %v", err)
	}
	defer onion.Close()

	for _, port := range config.TorListenConfig.RemotePorts {
		fmt.Printf("SOCKS5 server is listening on %v.onion:%v\n", onion.ID, port)
	}

	// Create a SOCKS5 server
	server, err := socks5.New(&config.SOCKS5Config)
	if err != nil {
		panic(err)
	}

	// Serve SOCKS5 over Tor
	if err := server.Serve(onion); err != nil {
		panic(err)
	}
}
