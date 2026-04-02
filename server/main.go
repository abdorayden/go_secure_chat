package main

import (
	"bufio"
	"encoding/base64"
	"log"
	"net"
	"strings"
	"sync"

	"chat_sec/server/enc"
)

const (
	Port = "9001"
)

type Client struct {
	Conn net.Conn
	Name string
	Addr string
}

var (
	clients    = make(map[string]*Client)
	clientsMu  sync.RWMutex
	serverPriv *enc.RSAPrivateKey
	serverPub  *enc.RSAPublicKey
	me         *Client
)

func handle(connection net.Conn) {
	senderAddr := connection.RemoteAddr().String()

	reader := bufio.NewReader(connection)

	_, err := connection.Write([]byte("PUBKEY:" + serverPub.Marshal() + "\n"))
	if err != nil {
		log.Printf("error sending public key to %s: %v", senderAddr, err)
		connection.Close()
		return
	}

	username, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("error reading username from %s: %v", senderAddr, err)
		connection.Close()
		return
	}
	username = strings.TrimSpace(strings.TrimPrefix(username, "USERNAME:"))
	if username == "" {
		username = "Anonymous"
	}

	client := &Client{
		Conn: connection,
		Name: username,
		Addr: senderAddr,
	}

	me = client

	clientsMu.Lock()
	clients[senderAddr] = client
	clientsMu.Unlock()

	broadcastSystemMessage(username + " joined the chat")

	defer func() {
		clientsMu.Lock()
		delete(clients, senderAddr)
		clientsMu.Unlock()
		connection.Close()
		broadcastSystemMessage(username + " left the chat")
	}()

	log.Printf("client connected: %s (%s)", username, senderAddr)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("read error from %s: %v", username, err)
			return
		}

		message = strings.TrimSpace(message)

		if message == "quit" {
			break
		}

		if !strings.HasPrefix(message, "MSG:") {
			continue
		}

		ciphertextB64 := strings.TrimPrefix(message, "MSG:")
		ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
		if err != nil {
			log.Printf("invalid ciphertext from %s: %v", username, err)
			continue
		}

		plaintext, err := enc.Decrypt(serverPriv, ciphertext)
		if err != nil {
			log.Printf("decryption error from %s: %v", username, err)
			continue
		}

		broadcastMsg := username + ": " + string(plaintext)
		broadcastMessage(broadcastMsg)
	}
}

func broadcastMessage(msg string) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	fullMsg := msg + "\n"
	for _, client := range clients {
		if client.Name != me.Name {
			if _, err := client.Conn.Write([]byte(fullMsg)); err != nil {
				log.Printf("write error to %s: %v", client.Name, err)
			}
		}
	}
}

func broadcastSystemMessage(msg string) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	fullMsg := "[System] " + msg + "\n"
	for _, client := range clients {
		if _, err := client.Conn.Write([]byte(fullMsg)); err != nil {
			log.Printf("write error to %s: %v", client.Name, err)
		}
	}
}

func main() {
	var err error

	serverPriv, serverPub, err = enc.GenerateKeyPair(2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key pair: %v", err)
	}
	log.Printf("RSA key pair generated (2048 bits)")

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
