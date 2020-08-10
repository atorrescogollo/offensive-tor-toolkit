package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	libtor "github.com/ipsn/go-libtor"
	"golang.org/x/net/html"
)

var creator = libtor.Creator

type LibTorWrapper struct{}

func (LibTorWrapper) New(ctx context.Context, args ...string) (process.Process, error) {
	return creator.New(ctx, args...)
}

func main() {
	// Start tor with some defaults + elevated verbosity
	fmt.Println("Starting tor instance, please wait a bit...")
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: LibTorWrapper{}, DebugWriter: os.Stderr})
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	defer t.Close()

	// Wait at most a few minutes to publish the service
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Make connection
	dialer, err := t.Dialer(ctx, nil)
	if err != nil {
		log.Fatal(err)
		return
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext}}
	// Get /
	resp, err := httpClient.Get("https://check.torproject.org")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()
	// Grab the <title>
	parsed, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Printf("Title: %v\n", getTitle(parsed))
	return
}

func getTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		var title bytes.Buffer
		if err := html.Render(&title, n.FirstChild); err != nil {
			panic(err)
		}
		return strings.TrimSpace(title.String())
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := getTitle(c); title != "" {
			return title
		}
	}
	return ""
}
