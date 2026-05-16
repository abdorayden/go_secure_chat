// Copyright (c) 2026 abdenour souane. All Rights Reserved.

// network package is a main component that used to handle connection between server and client,
// handshake, and more using a builtin protocol
// that is defined in separet package, and use builtin Packet implementation that is decode and encode json shared object
// between server and client
// this network package collect all components together

package network

import (
	"bufio"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	// my internal packages
	appcrypto "chat_sec/internal/crypto"
	"chat_sec/internal/protocol"
	"chat_sec/internal/storage"
)

// define constants
const (
	defaultWriteTimeout = 5 * time.Second
	defaultSendQueue    = 64
)

// Server struct
type Server struct {
	// server data
	addr       string           // addr of the server
	logger     *slog.Logger     // logger for more informations
	db         *storage.DBModel // the db model
	listener   net.Listener     // net tcp listener
	privateKey *rsa.PrivateKey  // private key (RSA)

	// clients with data
	mu      sync.RWMutex       // read write mutex for safety access to a global variable
	clients map[string]*Client // map of clients connected to the server

	// signals
	shutdown chan struct{}  // shutdown for signals (safety memory clean)
	wg       sync.WaitGroup // wait group of async tasks
}

// Client struct
type Client struct {
	Conn      net.Conn             // connection instance for the client
	Username  string               // client username
	Email     string               // client email
	AESKey    []byte               // client AES key
	Send      chan protocol.Packet // Send packet that used for communication
	PublicKey string               // public key (RSA)
	Addr      string               // addr of the server
}

// RunServer the entry point of lunching the server
// @param ctx context.Context the function context that used to handle unix signals in case we pressed CTRL-C
// @param addr string addr we are listening to
// @return error Error in case we have error
func RunServer(ctx context.Context, addr string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db, err := storage.Open(filepath.Join(".", "db.db"), filepath.Join(".", "migrations", "001_users.sql"))
	if err != nil {
		return err
	}
	defer db.Close()

	privateKey, err := appcrypto.GenerateRSAKeyPair(2048) // generate 2 keys
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server := &Server{
		addr:       addr,
		logger:     logger,
		db:         db,
		listener:   listener,
		privateKey: privateKey,
		clients:    make(map[string]*Client),
		shutdown:   make(chan struct{}),
	}

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-sigCtx.Done()
		server.logger.Info("shutdown requested")
		_ = server.Close()
	}()

	server.logger.Info("server listening", "addr", addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-server.shutdown: // in case server shutdown we wait for all clients
				server.wg.Wait()
				return nil
			default:
				server.logger.Error("accept failed", "error", err)
				continue
			}
		}
		server.wg.Add(1) // add one for wait group because we have client connected
		go func() {
			defer server.wg.Done()
			server.handleConn(sigCtx, conn)
		}()
	}
}

func (s *Server) Close() error {
	select {
	case <-s.shutdown:
	default:
		close(s.shutdown)
	}
	s.mu.Lock()
	for _, client := range s.clients {
		close(client.Send)
		_ = client.Conn.Close()
	}
	s.clients = map[string]*Client{}
	s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConn function it's a routine that is executed in separet thread for each client connected to the server
// @param ctx context.Context used to detect signal if the user exit or something
// @param conn net.Conn the connection of the client that is already connected
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {

	// this part is initializing the handshake

	addr := conn.RemoteAddr().String()
	s.logger.Info("connection opened", "addr", addr)
	defer func() {
		_ = conn.Close()
		s.logger.Info("connection closed", "addr", addr)
	}()

	reader := bufio.NewReader(conn)
	pubKeyRaw, err := appcrypto.MarshalPublicKey(&s.privateKey.PublicKey) // get the public key
	if err != nil {
		// problem with pair keys geterated
		s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeError, Error: "server key error"})
		return
	}
	if err := s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeHandshake, Payload: pubKeyRaw}); err != nil {
		return
	}

	packet, err := protocol.DecodeLine(reader) // handshake
	if err != nil || packet.Type != protocol.TypeHandshake {
		s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeError, Error: "invalid handshake"})
		return
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(packet.Payload)
	if err != nil {
		s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeError, Error: "invalid handshake payload"})
		return
	}

	aesKey, err := appcrypto.DecryptRSAOAEP(s.privateKey, encryptedKey)
	if err != nil {
		s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeError, Error: "handshake decryption failed"})
		return
	}

	client := &Client{
		Conn:   conn,
		AESKey: aesKey,
		Send:   make(chan protocol.Packet, defaultSendQueue),
		Addr:   addr,
	}

	go func() {
		// run it in separet thread
		s.writer(client)
	}()

	// handshake is okay
	// the tunnel is safe
	s.sendEncrypted(client, protocol.Packet{
		Type:    protocol.TypeSystem,
		Status:  "handshake_ok",
		Payload: "secure transport established",
	})

	for {
		select {
		case <-ctx.Done():
			s.unregister(client)
			return
		default:
		}

		packet, err := readEncryptedPacket(reader, client.AESKey)
		if err != nil {
			if client.Username != "" {
				s.unregister(client)
			}
			return
		}
		if err := s.handlePacket(ctx, client, packet); err != nil {
			s.sendEncrypted(client, protocol.Packet{Type: protocol.TypeError, Error: err.Error()})
		}
	}
}

func (s *Server) handlePacket(ctx context.Context, client *Client, packet protocol.Packet) error {
	switch packet.Type {
	case protocol.TypeAuthSignup:
		if strings.TrimSpace(packet.PublicKey) == "" {
			return errors.New("missing client public key")
		}
		if err := s.db.CreateUser(ctx, packet.Username, packet.Email, packet.Password); err != nil {
			return err
		}
		user, err := s.db.AuthenticateUser(ctx, packet.Email, packet.Password)
		if err != nil {
			return err
		}
		return s.finishAuth(client, user.Username, user.Email, packet.PublicKey)
	case protocol.TypeAuthLogin:
		if strings.TrimSpace(packet.PublicKey) == "" {
			return errors.New("missing client public key")
		}
		user, err := s.db.AuthenticateUser(ctx, packet.Email, packet.Password)
		if err != nil {
			return err
		}
		return s.finishAuth(client, user.Username, user.Email, packet.PublicKey)
	case protocol.TypeMessage:
		if client.Username == "" {
			return errors.New("authentication required")
		}
		if len(packet.Ciphertext) > protocol.MaxMessageLen*4 {
			return errors.New("message too large")
		}
		packet.Username = client.Username
		if err := s.db.SaveMessageHistory(ctx, packet.MessageID, client.Username, packet.Ciphertext, packet.EncryptedKeys, packet.Timestamp); err != nil {
			return err
		}
		s.broadcastMessage(client, packet)
		return nil
	default:
		return errors.New("unsupported packet type")
	}
}

func (s *Server) finishAuth(client *Client, username, email, publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[username]; exists {
		return storage.ErrDuplicateLogin
	}
	client.Username = username
	client.Email = email
	client.PublicKey = publicKey
	s.clients[username] = client
	s.sendEncrypted(client, protocol.Packet{
		Type:     protocol.TypeSystem,
		Status:   "auth_ok",
		Username: username,
		Payload:  fmt.Sprintf("authenticated as %s", username),
	})
	if err := s.sendHistoryLocked(context.Background(), client); err != nil {
		return err
	}
	s.sendEncrypted(client, protocol.Packet{
		Type:  protocol.TypePeerSync,
		Peers: s.peerSnapshotLocked(),
	})
	s.broadcastPeerSyncLocked()
	s.broadcastSystemLocked(fmt.Sprintf("%s joined the chat", username))
	return nil
}

func (s *Server) sendHistoryLocked(ctx context.Context, client *Client) error {
	records, err := s.db.LoadMessageHistoryForUser(ctx, client.Username, 200)
	if err != nil {
		return err
	}
	for _, record := range records {
		s.sendEncrypted(client, protocol.Packet{
			Type:       protocol.TypeMessage,
			Username:   record.SenderUsername,
			MessageID:  record.MessageID,
			Ciphertext: record.Ciphertext,
			EncryptedKeys: map[string]string{
				client.Username: record.EncryptedKey,
			},
			Timestamp: record.CreatedAtUnix,
			Status:    "history",
		})
	}
	return nil
}

func (s *Server) unregister(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if client.Username == "" {
		return
	}
	current, exists := s.clients[client.Username]
	if !exists || current != client {
		return
	}
	delete(s.clients, client.Username)
	close(client.Send)
	s.broadcastPeerSyncLocked()
	s.broadcastSystemLocked(fmt.Sprintf("%s left the chat", client.Username))
}

func (s *Server) broadcastMessage(sender *Client, packet protocol.Packet) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for username, recipient := range s.clients {
		if sender != nil && username == sender.Username {
			continue
		}
		if _, ok := packet.EncryptedKeys[username]; !ok {
			continue
		}
		out := packet
		out.EncryptedKeys = map[string]string{
			username: packet.EncryptedKeys[username],
		}
		s.sendEncrypted(recipient, out)
	}
}

func (s *Server) broadcastSystemLocked(message string) {
	packet := protocol.Packet{
		Type:      protocol.TypeSystem,
		Payload:   message,
		Timestamp: time.Now().Unix(),
	}
	for _, client := range s.clients {
		s.sendEncrypted(client, packet)
	}
}

func (s *Server) broadcastPeerSyncLocked() {
	packet := protocol.Packet{
		Type:  protocol.TypePeerSync,
		Peers: s.peerSnapshotLocked(),
	}
	for _, client := range s.clients {
		s.sendEncrypted(client, packet)
	}
}

func (s *Server) peerSnapshotLocked() []protocol.Peer {
	peers := make([]protocol.Peer, 0, len(s.clients))
	for _, client := range s.clients {
		peers = append(peers, protocol.Peer{
			Username:  client.Username,
			PublicKey: client.PublicKey,
		})
	}
	return peers
}

// writer is a method of Server struct
// that accept the client to write encrypted packet into Packet channel in Server
func (s *Server) writer(client *Client) {
	for packet := range client.Send {
		if err := writeEncryptedPacket(client.Conn, client.AESKey, packet); err != nil {
			return
		}
	}
}

func (s *Server) sendEncrypted(client *Client, packet protocol.Packet) {
	select {
	case client.Send <- packet:
	default:
		s.logger.Error("dropping packet due to full send queue", "username", client.Username)
	}
}

func (s *Server) writePlainPacket(conn net.Conn, packet protocol.Packet) error {
	_ = conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	defer conn.SetWriteDeadline(time.Time{})
	return protocol.Encode(conn, packet)
}

func writeEncryptedPacket(conn net.Conn, key []byte, packet protocol.Packet) error {
	body, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	encrypted, err := appcrypto.EncryptBase64(key, body)
	if err != nil {
		return err
	}
	envelope := protocol.Packet{
		Type:      protocol.TypeSystem,
		Status:    "transport",
		Payload:   encrypted,
		Timestamp: time.Now().Unix(),
	}
	_ = conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	defer conn.SetWriteDeadline(time.Time{})
	return protocol.Encode(conn, envelope)
}

func readEncryptedPacket(reader *bufio.Reader, key []byte) (protocol.Packet, error) {
	envelope, err := protocol.DecodeLine(reader)
	if err != nil {
		return protocol.Packet{}, err
	}
	if strings.TrimSpace(envelope.Payload) == "" {
		return protocol.Packet{}, errors.New("missing encrypted payload")
	}
	body, err := appcrypto.DecryptBase64(key, envelope.Payload)
	if err != nil {
		return protocol.Packet{}, err
	}
	var packet protocol.Packet
	if err := json.Unmarshal(body, &packet); err != nil {
		return protocol.Packet{}, err
	}
	if err := packet.Validate(); err != nil {
		return protocol.Packet{}, err
	}
	return packet, nil
}
