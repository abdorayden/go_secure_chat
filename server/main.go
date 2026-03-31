package main

// TODO: handle the auth key
// TODO: use enc from scratch (RSA)

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
)

const (
	Port = "9001"
)

var (
	broadcast   = make(map[string]net.Conn)
	broadcastMu sync.RWMutex
)

func handle(connection net.Conn) {
	senderAddr := connection.RemoteAddr().String()

	defer func() {
		broadcastMu.Lock()
		delete(broadcast, senderAddr)
		broadcastMu.Unlock()
		if err := connection.Close(); err != nil {
			log.Printf("error closing connection from client %v: %v", connection.RemoteAddr(), err)
		}
	}()

	reader := bufio.NewReader(connection)

	broadcastMu.Lock()
	broadcast[senderAddr] = connection
	broadcastMu.Unlock()

	log.Printf("client connected: %s", senderAddr)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("read error from %s: %v", senderAddr, err)
			return
		}

		if strings.TrimSpace(message) == "quit" {
			break
		}

		messageByte := []byte(message)

		broadcastMu.RLock()
		for addr, conn := range broadcast {
			if addr == senderAddr {
				continue
			}
			if _, err := conn.Write(messageByte); err != nil {
				log.Printf("write error to %s: %v", addr, err)
				broadcastMu.RUnlock()
				broadcastMu.Lock()
				delete(broadcast, addr)
				broadcastMu.Unlock()
				broadcastMu.RLock()
			}
		}
		broadcastMu.RUnlock()
	}
}

func main() {
	var err error

	listener, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Fatalf("cannot listen on port %v: %v", Port, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Printf("error closing listener: %v", err)
		}
	}()

	log.Printf("listening on port %v", Port)

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handle(connection)
	}
}
