package main

import (
	"context"
	"flag"
	"log"

	"chat_sec/internal/network"
)

func main() {
	addr := flag.String("addr", ":9001", "server listen address")
	flag.Parse()

	if err := network.RunServer(context.Background(), *addr); err != nil {
		log.Fatal(err)
	}
}
