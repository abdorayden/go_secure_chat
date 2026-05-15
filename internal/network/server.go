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

	appcrypto "chat_sec/internal/crypto"
	"chat_sec/internal/protocol"
	"chat_sec/internal/storage"
)

const (
	defaultWriteTimeout = 5 * time.Second
	defaultSendQueue    = 64
)

type Server struct {
	addr       string
	logger     *slog.Logger
	db         *storage.DBModel
	listener   net.Listener
	privateKey *rsa.PrivateKey

	mu      sync.RWMutex
	clients map[string]*Client

	shutdown chan struct{}
	wg       sync.WaitGroup
}

type Client struct {
	Conn      net.Conn
	Username  string
	Email     string
	AESKey    []byte
	Send      chan protocol.Packet
	PublicKey string
	Addr      string
}

func RunServer(ctx context.Context, addr string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db, err := storage.Open(filepath.Join(".", "db.db"), filepath.Join(".", "migrations", "001_users.sql"))
	if err != nil {
		return err
	}
	defer db.Close()

	privateKey, err := appcrypto.GenerateRSAKeyPair(2048)
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
			case <-server.shutdown:
				server.wg.Wait()
				return nil
			default:
				server.logger.Error("accept failed", "error", err)
				continue
			}
		}
		server.wg.Add(1)
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

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	addr := conn.RemoteAddr().String()
	s.logger.Info("connection opened", "addr", addr)
	defer func() {
		_ = conn.Close()
		s.logger.Info("connection closed", "addr", addr)
	}()

	reader := bufio.NewReader(conn)
	pubKeyRaw, err := appcrypto.MarshalPublicKey(&s.privateKey.PublicKey)
	if err != nil {
		s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeError, Error: "server key error"})
		return
	}
	if err := s.writePlainPacket(conn, protocol.Packet{Type: protocol.TypeHandshake, Payload: pubKeyRaw}); err != nil {
		return
	}

	packet, err := protocol.DecodeLine(reader)
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
		s.writer(client)
	}()

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
	s.sendEncrypted(client, protocol.Packet{
		Type:  protocol.TypePeerSync,
		Peers: s.peerSnapshotLocked(),
	})
	s.broadcastPeerSyncLocked()
	s.broadcastSystemLocked(fmt.Sprintf("%s joined the chat", username))
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
