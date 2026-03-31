package main

// TODO: add auth token to join the club
// TODO: secure with encryptio use(RSA)

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
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
	broadcast   map[string]net.Conn
	broadcastMu sync.Mutex
)

func handle(connection net.Conn) {
	var err error
	senderAddr := connection.RemoteAddr().String()
	defer func() {
		broadcastMu.Lock()
		delete(broadcast, senderAddr)
		broadcastMu.Unlock()
		if connection.Close() != nil {
			log.Fatalf("cannot close a tcp from client %v", connection.RemoteAddr().String())
		}
	}()

	broadcastMu.Lock()
	broadcast[senderAddr] = connection
	broadcastMu.Unlock()

	reader := bufio.NewReader(connection)
	for {
		var message string
		message, err = reader.ReadString('\n')

		if err != nil {
			if message == "" {
				return
			}
			log.Println(err)
		}

		if strings.TrimSpace(message) == "quit" {
			break
		}

		broadcastMu.Lock()
		for ip_addr, conn := range broadcast {
			if ip_addr == senderAddr {
				continue
			}
			var n int
			n, err := conn.Write([]byte(message))
			if err != nil {
				log.Println(err)
			}

			if n == 0 {
				log.Println("client quit")
			}
		}
		broadcastMu.Unlock()

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
	broadcast = make(map[string]net.Conn)

	for {
		connection, err = listener.Accept()
		if err != nil {
			log.Fatal("cannot accept")
		}
		go handle(connection)
	}
}
