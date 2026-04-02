package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"chat_sec/server/enc"
)

const (
	ServerPort = "9001"
	ServerHost = "localhost"
)

func main() {
	fmt.Print("Enter your username: ")
	username, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		username = "Anonymous"
	}

	connection, err := net.Dial("tcp", ServerHost+":"+ServerPort)
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.Close()

	reader := bufio.NewReader(connection)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalln("failed to receive public key:", err)
	}

	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "PUBKEY:") {
		log.Fatalln("unexpected server message:", line)
	}

	pubKeyStr := strings.TrimPrefix(line, "PUBKEY:")
	serverPub, err := enc.UnmarshalPublicKey(pubKeyStr)
	if err != nil {
		log.Fatalln("failed to parse public key:", err)
	}
	log.Println("Received server public key")

	_, err = connection.Write([]byte("USERNAME:" + username + "\n"))
	if err != nil {
		log.Fatalln("failed to send username:", err)
	}
	log.Printf("Joined as: %s", username)

	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				log.Println("server disconnected")
				return
			}
			log.Print(strings.TrimSpace(msg))
		}
	}()

	stdin := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, err := stdin.ReadString('\n')
		if err != nil {
			break
		}
		text = strings.TrimSpace(text)
		if text == "quit" {
			break
		}
		if text == "" {
			continue
		}

		ciphertext, err := enc.Encrypt(serverPub, []byte(text))
		if err != nil {
			log.Println("encryption error:", err)
			continue
		}

		ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
		_, err = connection.Write([]byte("MSG:" + ciphertextB64 + "\n"))
		if err != nil {
			log.Println("send error:", err)
			break
		}
	}
}
