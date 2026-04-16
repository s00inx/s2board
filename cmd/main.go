package main

import (
	"flag"
	"log"
	"os"
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

	app := internal.NewApp(cfg)

	// 1. Запускаем сервер в фоне
	go app.Run()

	// 2. Настраиваем канал для перехвата сигналов (Ctrl+C и т.д.)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 3. БЛОКИРУЕМ main здесь. Программа будет стоять и ждать, пока в канал придет сигнал
	sig := <-sigChan
	log.Printf("Received signal: %v. Shutting down...", sig)

	// 4. После того как сигнал пришел — выполняем завершающие действия
	// Используем http.Client{} напрямую или создаем с таймаутом, чтобы не висеть вечно
	// app.Node.NodeBye(http.Client{})

}
