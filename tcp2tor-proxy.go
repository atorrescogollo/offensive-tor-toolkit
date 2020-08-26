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
	TorConfig    tor.StartConf
	Timeout      int
	Listen       string
	OnionForward string
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
	flag.StringVar(&config.Listen, "listen", "127.0.0.1:60101", "TCP Socket to listen on. Format: [<IP>]:<PORT>")
	flag.StringVar(&config.OnionForward, "onion-forward", "", "Hidden service to proxy. Format: <ONION>:<PORT>. This parameter is required")
	flag.IntVar(&config.Timeout, "timeout", 180, "Timeout in seconds for Tor setup")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	if config.OnionForward == "" {
		fmt.Fprintf(os.Stderr, "OnionForward parameter (-onion-forward) is required.\n\n")
		flag.Usage()
		os.Exit(1)
	}
	return config
}

func main() {
	config := parseArgs()
	// Start tor with some defaults
	fmt.Println("Starting tor instance, please wait a bit...")
	t, err := tor.Start(nil, &config.TorConfig)
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	defer t.Close()

	// Wait at most a few minutes
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	// Make connection
	dialer, err := t.Dialer(ctx, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Listen TCP
	listener, err := net.Listen("tcp", config.Listen)
	if err != nil {
		panic("connection error:" + err.Error())
	}

	fmt.Printf("Proxying %v -> %v\n", config.Listen, config.OnionForward)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept Error:", err)
			continue
		}

		go handleRequest(dialer, conn, config.OnionForward)
	}
}

func handleRequest(dialer *tor.Dialer, conn net.Conn, host string) {
	fmt.Println("New Connection.")

	// Connect to hidden service
	proxy, err := dialer.Dial("tcp", host)
	if err != nil {
		log.Panicf("Failed to forward to "+host+"%v", err)
		return
	}

	go copyIO(proxy, conn)
	go copyIO(conn, proxy)
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
