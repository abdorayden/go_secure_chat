// Copyright (c) 2026 abdenour souane. All Rights Reserved.

// network package is a main component that used to handle connection between server and client,
// handshake, and more using a builtin protocol
// that is defined in separet package, and use builtin Packet implementation that is decode and encode json shared object
// between server and client
// this network package collect all components together

package network

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appcrypto "chat_sec/internal/crypto"
	"chat_sec/internal/protocol"
)

type TransportClient struct {
	addr       string
	conn       net.Conn
	reader     *bufio.Reader
	aesKey     []byte
	privateKey *rsa.PrivateKey
	publicKey  string

	mu      sync.RWMutex
	peers   map[string]*rsa.PublicKey
	sendMu  sync.Mutex
	closing chan struct{}
}

func Dial(addr string) (*TransportClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(conn)
	hello, err := protocol.DecodeLine(reader)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if hello.Type != protocol.TypeHandshake {
		_ = conn.Close()
		return nil, errors.New("missing server handshake")
	}
	serverPub, err := appcrypto.ParsePublicKey(hello.Payload)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	aesKey, err := appcrypto.GenerateAESKey(32)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	encryptedKey, err := appcrypto.EncryptRSAOAEP(serverPub, aesKey)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := protocol.Encode(conn, protocol.Packet{
		Type:    protocol.TypeHandshake,
		Payload: base64.StdEncoding.EncodeToString(encryptedKey),
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	client := &TransportClient{
		addr:    addr,
		conn:    conn,
		reader:  reader,
		aesKey:  aesKey,
		peers:   make(map[string]*rsa.PublicKey),
		closing: make(chan struct{}),
	}
	return client, nil
}

func (c *TransportClient) PublicKey() string {
	return c.publicKey
}

func (c *TransportClient) Close() error {
	select {
	case <-c.closing:
	default:
		close(c.closing)
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *TransportClient) SendAuth(packet protocol.Packet) error {
	if err := c.EnsureIdentityKey(packet.Email); err != nil {
		return err
	}
	packet.PublicKey = c.publicKey
	return c.sendPacket(packet)
}

func (c *TransportClient) EnsureIdentityKey(accountID string) error {
	accountID = strings.TrimSpace(strings.ToLower(accountID))
	if accountID == "" {
		return errors.New("account id is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.privateKey != nil && c.publicKey != "" {
		return nil
	}

	keyPath := identityKeyPath(accountID)
	privateKey, err := loadOrCreateIdentityKey(keyPath)
	if err != nil {
		return err
	}
	publicKey, err := appcrypto.MarshalPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	c.privateKey = privateKey
	c.publicKey = publicKey
	return nil
}

func (c *TransportClient) sendPacket(packet protocol.Packet) error {
	body, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	encrypted, err := appcrypto.EncryptBase64(c.aesKey, body)
	if err != nil {
		return err
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	defer c.conn.SetWriteDeadline(time.Time{})
	return protocol.Encode(c.conn, protocol.Packet{
		Type:      protocol.TypeSystem,
		Status:    "transport",
		Payload:   encrypted,
		Timestamp: time.Now().Unix(),
	})
}

func (c *TransportClient) ReadPacket() (protocol.Packet, error) {
	return readEncryptedPacket(c.reader, c.aesKey)
}

func (c *TransportClient) UpdatePeers(peers []protocol.Peer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := make(map[string]*rsa.PublicKey, len(peers))
	for _, peer := range peers {
		pub, err := appcrypto.ParsePublicKey(peer.PublicKey)
		if err != nil {
			return err
		}
		next[peer.Username] = pub
	}
	c.peers = next
	return nil
}

func (c *TransportClient) EncryptMessage(sender string, plaintext string) (protocol.Packet, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if strings.TrimSpace(plaintext) == "" {
		return protocol.Packet{}, errors.New("empty message")
	}
	messageKey, err := appcrypto.GenerateAESKey(32)
	if err != nil {
		return protocol.Packet{}, err
	}
	ciphertext, err := appcrypto.EncryptBase64(messageKey, []byte(plaintext))
	if err != nil {
		return protocol.Packet{}, err
	}
	encryptedKeys := make(map[string]string, len(c.peers))
	for username, pub := range c.peers {
		wrapped, err := appcrypto.EncryptRSAOAEP(pub, messageKey)
		if err != nil {
			return protocol.Packet{}, err
		}
		encryptedKeys[username] = base64.StdEncoding.EncodeToString(wrapped)
	}
	if len(encryptedKeys) == 0 {
		return protocol.Packet{}, errors.New("no recipients available")
	}
	return protocol.Packet{
		Type:          protocol.TypeMessage,
		Username:      sender,
		MessageID:     fmtMessageID(),
		Ciphertext:    ciphertext,
		EncryptedKeys: encryptedKeys,
		Timestamp:     time.Now().Unix(),
	}, nil
}

func (c *TransportClient) SendMessage(packet protocol.Packet) error {
	return c.sendPacket(packet)
}

func (c *TransportClient) DecryptMessage(packet protocol.Packet, username string) (string, error) {
	c.mu.RLock()
	privateKey := c.privateKey
	c.mu.RUnlock()
	if privateKey == nil {
		return "", errors.New("identity key not initialized")
	}
	wrapped, ok := packet.EncryptedKeys[username]
	if !ok {
		return "", errors.New("message not addressed to user")
	}
	wrappedBytes, err := base64.StdEncoding.DecodeString(wrapped)
	if err != nil {
		return "", err
	}
	messageKey, err := appcrypto.DecryptRSAOAEP(privateKey, wrappedBytes)
	if err != nil {
		return "", err
	}
	plaintext, err := appcrypto.DecryptBase64(messageKey, packet.Ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func fmtMessageID() string {
	return strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "_")
}

func identityKeyPath(accountID string) string {
	sum := sha256.Sum256([]byte(accountID))
	filename := hex.EncodeToString(sum[:]) + ".pem"
	return filepath.Join(".", "client_keys", filename)
}

func loadOrCreateIdentityKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return appcrypto.ParsePrivateKeyPEM(data)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	privateKey, err := appcrypto.GenerateRSAKeyPair(2048)
	if err != nil {
		return nil, err
	}
	pemData, err := appcrypto.MarshalPrivateKeyPEM(privateKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		return nil, err
	}
	return privateKey, nil
}
