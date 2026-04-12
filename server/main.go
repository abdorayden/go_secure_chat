package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"hash"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

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
	serverAES  []byte
	encMode    string
	me         *Client
)

func handle(connection net.Conn) {
	senderAddr := connection.RemoteAddr().String()

	reader := bufio.NewReader(connection)

	switch encMode {
	case "rsa":
		_, err := connection.Write([]byte("PUBKEY:" + serverPub.Marshal() + "\n"))
		if err != nil {
			log.Printf("error sending public key to %s: %v", senderAddr, err)
			connection.Close()
			return
		}
	case "aes":
		keyB64 := base64.StdEncoding.EncodeToString(serverAES)
		_, err := connection.Write([]byte("AESKEY:" + keyB64 + "\n"))
		if err != nil {
			log.Printf("error sending AES key to %s: %v", senderAddr, err)
			connection.Close()
			return
		}
	default:
		log.Printf("unknown encryption mode: %s", encMode)
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
		var plaintext []byte
		switch encMode {
		case "rsa":
			ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
			if err != nil {
				log.Printf("invalid ciphertext from %s: %v", username, err)
				continue
			}
			plaintext, err = enc.Decrypt(serverPriv, ciphertext)
			if err != nil {
				log.Printf("decryption error from %s: %v", username, err)
				continue
			}
		case "aes":
			plaintext, err = enc.DecryptAESBase64(serverAES, ciphertextB64)
			if err != nil {
				log.Printf("decryption error from %s: %v", username, err)
				continue
			}
		default:
			log.Printf("unknown encryption mode: %s", encMode)
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
		if _, err := client.Conn.Write([]byte(fullMsg)); err != nil {
			log.Printf("write error to %s: %v", client.Name, err)
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

const DBFile = "./db.db"

type DBModel struct {
	db *sql.DB
}

func newDBModel(db *sql.DB) (*DBModel, error) {
	// db
	var fileContentAsBytes []byte
	var fileContent string
	var err error

	db, err = sql.Open("sqlite3", DBFile)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = db.Close()
		if err != nil {
			panic(err) // no option available
		}
	}()

	fileContentAsBytes, err = os.ReadFile("./queries/tables.sql")
	if err != nil {
		return nil, err
	}

	// NOTE: succefully load tables
	fileContent = string(fileContentAsBytes)
	_, err = db.Exec(fileContent)
	if err != nil {
		return nil, err
	}

	return &DBModel{
		db}, nil
}

var ErrUserExists = errors.New("user already exists")

func (dbmodel *DBModel) execQ(query string) bool {
	var (
		err  error
		rows *sql.Rows
	)
	rows, err = dbmodel.db.Query(query, nil)
	defer func() {
		err = rows.Close()
		if err != nil {
			panic(err)
		}
	}()

	return rows.Next()
}

func (dbmodel *DBModel) checkUser(mail, password string) bool {
	var query string
	pass := md5.New()
	hashedPassword := pass.Sum([]byte(password))
	query = fmt.Sprintf("SELECT * FROM users WHERE mail = %v and password = %v;", mail, hashedPassword)
	return dbmodel.execQ(query)
}

func (dbmodel *DBModel) signUp(username, mail, password string) (bool, error) {
	if dbmodel.checkUser(mail, password) {
		return false, ErrUserExists
	}
	var (
		query string
		pass  hash.Hash
	)
	pass = md5.New()
	hashedPassword := pass.Sum([]byte(password))
	query = fmt.Sprintf("insert into users values (%v , %v , %v);", username, hashedPassword, mail)
	return dbmodel.execQ(query), nil
}

func main() {

	var (
		err        error
		aesKeyB64  string
		aesKeySize int
		db         *sql.DB
		dbModel    *DBModel
	)

	dbModel, err = newDBModel(db)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: add button to let user to reconnect to the server

	// TODO: get signin and signup data to manage the db after prepare queries for them
	// TODO: handle format for signin and signup for the server
	_ = dbModel

	flag.StringVar(&encMode, "enc", "rsa", "encryption mode: rsa or aes")
	flag.StringVar(&aesKeyB64, "aes-key", "", "base64-encoded AES key (optional)")
	flag.IntVar(&aesKeySize, "aes-key-size", 32, "AES key size in bytes when generating (16, 24, 32)")
	flag.Parse()

	encMode = strings.ToLower(encMode)
	if encMode != "rsa" && encMode != "aes" {
		log.Fatalf("invalid -enc value %q (use rsa or aes)", encMode)
	}

	switch encMode {
	case "rsa":
		serverPriv, serverPub, err = enc.GenerateKeyPair(2048)
		if err != nil {
			log.Fatalf("failed to generate RSA key pair: %v", err)
		}
		log.Printf("RSA key pair generated (2048 bits)")
	case "aes":
		if aesKeyB64 != "" {
			serverAES, err = base64.StdEncoding.DecodeString(aesKeyB64)
			if err != nil {
				log.Fatalf("invalid -aes-key base64: %v", err)
			}
			if len(serverAES) != 16 && len(serverAES) != 24 && len(serverAES) != 32 {
				log.Fatalf("invalid AES key length %d (must be 16, 24, or 32 bytes)", len(serverAES))
			}
		} else {
			serverAES, err = enc.GenerateAESKey(aesKeySize)
			if err != nil {
				log.Fatalf("failed to generate AES key: %v", err)
			}
		}
		log.Printf("AES mode enabled (key size %d bytes)", len(serverAES))
	}

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
