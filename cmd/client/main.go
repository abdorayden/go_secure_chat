package main

import (
	"flag"
	"log"
	"os"

	"gioui.org/app"

	"chat_sec/internal/ui"
)

func main() {
	addr := flag.String("addr", "localhost:9001", "server address")
	flag.Parse()

	go func() {
		if err := ui.Run(*addr); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
