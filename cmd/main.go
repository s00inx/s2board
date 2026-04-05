package main

import (
	"github.com/s00inx/stdesk/internal"
)

func main() {
	cfg := &internal.Config{
		DataDir: "data/",
		Port:    "8080",
	}
	app := internal.NewApp(cfg)
	app.Run()
}
