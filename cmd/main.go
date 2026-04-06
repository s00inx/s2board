package main

import (
	"flag"

	"github.com/s00inx/s2board/internal"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	dataDir := flag.String("dir", "./data", "Directory for database and blobs")
	nodeName := flag.String("name", "MyNode", "Optional node name")
	flag.Parse()

	// 2. Создаем конфиг
	cfg := &internal.Config{
		DataDir: *dataDir,
		Port:    *port,
		Name:    *nodeName,
	}

	// 3. Инициализируем и запускаем приложение
	app := internal.NewApp(cfg)
	app.Run()
}
