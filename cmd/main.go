package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

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

	// create context for shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := internal.NewApp(cfg)

	go app.Run(ctx)

	<-ctx.Done()
	log.Println("Shutting down... waiting for goroutines")

	// waiting for all exit proccesses done
	app.Wait()
	log.Println("Exit.")
}
