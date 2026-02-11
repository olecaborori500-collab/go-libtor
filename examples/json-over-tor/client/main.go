package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

type payload struct {
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

func main() {
	var dataDir, url, message, source string
	flag.StringVar(&dataDir, "data-dir", "./tor-data-client", "tor data directory")
	flag.StringVar(&url, "url", "", "destination url, e.g. http://<onion>.onion/json")
	flag.StringVar(&message, "message", "hello from go-libtor", "message field for JSON payload")
	flag.StringVar(&source, "source", "go-libtor-client", "source field for JSON payload")
	flag.Parse()

	if url == "" {
		log.Fatal("-url is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("starting embedded tor for JSON sender...")
	t, err := tor.Start(nil, &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DataDir:        dataDir,
		DebugWriter:    os.Stderr,
	})
	if err != nil {
		log.Fatalf("failed to start tor: %v", err)
	}
	defer t.Close()

	dialer, err := t.Dialer(ctx, nil)
	if err != nil {
		log.Fatalf("failed to create tor dialer: %v", err)
	}

	transport := &http.Transport{DialContext: dialer.DialContext}
	client := &http.Client{Transport: transport, Timeout: 90 * time.Second}

	body, err := json.Marshal(payload{Message: message, Source: source})
	if err != nil {
		log.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to send request over tor: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("response status=%s body=%s", resp.Status, string(respBody))
}
