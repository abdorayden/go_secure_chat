package main

import (
	"log"
	"net"
)

const Port = "9001"

func main() {
	var listener net.Listener
	var connection net.Conn
	var err error
	listener, err = net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Fatalf("cannot listen to a tcp on port %v", Port)
	}

	defer func() {
		if listener.Close() != nil {
			log.Fatal("cannot close a tcp socket")
		}
	}()

	for {
		connection, err = listener.Accept()
		if err != nil {
			log.Fatal("cannot accept")
		}
		var n int
		n, err = connection.Write([]byte("hello from the server"))
		if err != nil {
			log.Fatal("cannot write")
		}
		log.Println(n)
	}

}
