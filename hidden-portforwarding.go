package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
)

type Config struct {
	TorConfig       tor.StartConf
	TorListenConfig tor.ListenConf
	Timeout         int
	HiddenPort      int
	Forward         string
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
	var flagHiddenSrvPort int
	flag.IntVar(&flagHiddenSrvPort, "hidden-port", 80, "Port for onion service")
	flag.StringVar(&config.Forward, "forward", "", "Where the hidden service should forward packets (local port forwarding). Format: <FW_IP>:<FW_PORT>. This parameter is required")
	flag.IntVar(&config.Timeout, "timeout", 180, "Timeout in seconds for Tor setup")
	config.TorListenConfig = tor.ListenConf{}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()
	if config.Forward == "" {
		fmt.Fprintf(os.Stderr, "Forward parameter (-forward) is required.\n\n")
		flag.Usage()
		os.Exit(1)
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

	// Create an onion service to listen on any port but show as 80
	onion, err := t.Listen(ctx, &config.TorListenConfig)
	if err != nil {
		log.Panicf("Failed to create onion service: %v", err)
	}
	defer onion.Close()

	for _, port := range config.TorListenConfig.RemotePorts {
		fmt.Printf("Forwarding %v.onion:%v -> %v\n", onion.ID, port, config.Forward)
	}

	for {
		conn, err := onion.Accept()
		if err != nil {
			log.Panicf("Error: %v", err)
			continue
		}

		go handleRequest(conn, config.Forward)
	}
}

func handleRequest(conn net.Conn, host string) {
	fmt.Println("New Connection.")

	proxy, err := net.Dial("tcp", host)
	if err != nil {
		log.Panicf("Failed to forward to "+host+"%v", err)
		return
	}
	go copyIO(conn, proxy)
	go copyIO(proxy, conn)
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
