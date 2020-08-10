package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/armon/go-socks5"
	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
)

var creator = libtor.Creator

type LibTorWrapper struct{}

func (LibTorWrapper) New(ctx context.Context, args ...string) (process.Process, error) {
	return creator.New(ctx, args...)
}

func main() {
	// Start tor with some defaults + elevated verbosity
	fmt.Println("Starting and registering onion service, please wait a bit...")
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: LibTorWrapper{}, DebugWriter: os.Stderr})
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	defer t.Close()

	// Wait at most a few minutes to publish the service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Create an onion service to listen on any port but show as 80
	onion, err := t.Listen(ctx, &tor.ListenConf{RemotePorts: []int{80}})
	if err != nil {
		log.Panicf("Failed to create onion service: %v", err)
	}
	defer onion.Close()

	fmt.Printf("SOCKS5 server is listening on %v.onion:80\n", onion.ID)

	// Create a SOCKS5 server
	cred := socks5.StaticCredentials{"hiddenuser": "hiddenandsupersecretpassword"}
	conf := &socks5.Config{Credentials: cred}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	// Serve SOCKS5 over Tor
	if err := server.Serve(onion); err != nil {
		panic(err)
	}
}
