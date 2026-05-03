package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/s00inx/s2board/internal"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	dataDir := flag.String("dir", "./data", "Directory for database and blobs")
	nodeName := flag.String("name", "MyNode", "Optional node name")
	flag.Parse()

	cfg := &internal.Config{
		DataDir: *dataDir,
		Port:    *port,
		Name:    *nodeName,
	}

	app := internal.NewApp(cfg)

	go app.Run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	app.Node.Byew()

	log.Printf("Received signal: %v. Shutting down...", sig)
	time.Sleep(200 * time.Millisecond)
}
