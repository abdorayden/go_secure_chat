package main

// TODO: add auth token to join the club
// TODO: secure with encryptio use(RSA)

import (
	"bufio"
	"log"
	"net"
)

const Port = "9001"

type Client struct {
	ip_addr    string
	connection net.Conn
}

func NewClient(connection net.Conn) *Client {
	return &Client{
		connection: connection,
		ip_addr:    connection.RemoteAddr().String(),
	}
}

var (
	broadcast map[string]net.Conn
)

func handle(connection net.Conn) {
	var err error
	defer func() {
		if connection.Close() != nil {
			log.Fatalf("cannot close a tcp from client %v", connection.RemoteAddr().String())
		}
	}()

	broadcast[connection.RemoteAddr().String()] = connection

	reader := bufio.NewReader(connection)
	for {
		var message string
		message, err = reader.ReadString('\n')

		if err != nil {
			log.Println(err)
		}

		if len(message) > 0 && message == "quit" {
			break
		}

		for ip_addr, conn := range broadcast {
			if ip_addr != conn.RemoteAddr().String() {
				var n int
				n, err := connection.Write([]byte(message))
				if err != nil {
					log.Println(err)
				}

				if n == 0 {
					log.Println("client quit")
					break
				}
			}
		}

	}
}

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

	log.Printf("listining on port %v", Port)

	for {
		connection, err = listener.Accept()
		if err != nil {
			log.Fatal("cannot accept")
		}
		go handle(connection)
	}
}
