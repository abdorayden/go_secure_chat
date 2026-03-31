package main

import (
	"log"
	"net"
)

const (
	ServerPort = "9001"
	ServerHost = "localhost"
)

func main() {

	var connection net.Conn
	var err error
	var n int

	connection, err = net.Dial("tcp", ServerHost+":"+ServerPort)

	if err != nil {
		log.Fatalln(err)
	}

	n, err = connection.Write([]byte("hello"))

	if err != nil {
		log.Fatalln(err)
	}
	log.Print(n)
}
