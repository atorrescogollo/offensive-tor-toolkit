package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

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

	fmt.Printf("Please open a Tor capable netcat and navigate to %v.onion:80\n", onion.ID)

	for {
		conn, err := onion.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(os.Stdin)
	var buf [512]byte

	fmt.Println("Client connected!")
	for {
		data, _ := reader.ReadString('\n')
		_, err2 := conn.Write([]byte(data))
		if err2 != nil {
			return
		}
		n, err := conn.Read(buf[0:])
		if err != nil {
			return
		}
		received := string(buf[0:n])
		fmt.Println(received)
	}
}
