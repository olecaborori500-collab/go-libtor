package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

type payload struct {
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

func main() {
	var dataDir string
	var remotePort int
	flag.StringVar(&dataDir, "data-dir", "./tor-data-server", "tor data directory")
	flag.IntVar(&remotePort, "remote-port", 80, "onion service remote port")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("starting embedded tor for JSON receiver...")
	t, err := tor.Start(nil, &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DataDir:        dataDir,
		DebugWriter:    os.Stderr,
	})
	if err != nil {
		log.Fatalf("failed to start tor: %v", err)
	}
	defer t.Close()

	pubCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	onion, err := t.Listen(pubCtx, &tor.ListenConf{RemotePorts: []int{remotePort}})
	if err != nil {
		log.Fatalf("failed to create onion service: %v", err)
	}
	defer onion.Close()

	log.Printf("json endpoint: http://%s.onion/json", onion.ID)

	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body payload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
			return
		}
		log.Printf("received payload: message=%q source=%q", body.Message, body.Source)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	if err := srv.Serve(onion); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
}
