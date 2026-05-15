package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

const (
	TypeHandshake  = "handshake"
	TypeAuthSignup = "auth_signup"
	TypeAuthLogin  = "auth_login"
	TypeMessage    = "message"
	TypeSystem     = "system"
	TypeError      = "error"
	TypePeerSync   = "peer_sync"
)

const MaxMessageLen = 4096

var ErrMalformedPacket = errors.New("malformed packet")

type Packet struct {
	Type          string            `json:"type"`
	Username      string            `json:"username,omitempty"`
	Email         string            `json:"email,omitempty"`
	Password      string            `json:"password,omitempty"`
	Payload       string            `json:"payload,omitempty"`
	Timestamp     int64             `json:"timestamp,omitempty"`
	PublicKey     string            `json:"public_key,omitempty"`
	MessageID     string            `json:"message_id,omitempty"`
	Ciphertext    string            `json:"ciphertext,omitempty"`
	EncryptedKeys map[string]string `json:"encrypted_keys,omitempty"`
	Status        string            `json:"status,omitempty"`
	Error         string            `json:"error,omitempty"`
	Peers         []Peer            `json:"peers,omitempty"`
}

type Peer struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}

func (p *Packet) Normalize() {
	if p.Timestamp == 0 {
		p.Timestamp = time.Now().Unix()
	}
}

func (p Packet) Validate() error {
	if strings.TrimSpace(p.Type) == "" {
		return ErrMalformedPacket
	}
	switch p.Type {
	case TypeHandshake:
		if strings.TrimSpace(p.Payload) == "" {
			return ErrMalformedPacket
		}
	case TypeAuthSignup, TypeAuthLogin:
		if strings.TrimSpace(p.Email) == "" || strings.TrimSpace(p.Password) == "" {
			return ErrMalformedPacket
		}
	case TypeMessage:
		if strings.TrimSpace(p.MessageID) == "" || strings.TrimSpace(p.Ciphertext) == "" {
			return ErrMalformedPacket
		}
		if len(p.EncryptedKeys) == 0 {
			return ErrMalformedPacket
		}
	case TypeSystem, TypeError, TypePeerSync:
	default:
		return ErrMalformedPacket
	}
	return nil
}

func Encode(w io.Writer, packet Packet) error {
	packet.Normalize()
	if err := packet.Validate(); err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(packet)
}

func Decode(r io.Reader) (Packet, error) {
	var packet Packet
	err := json.NewDecoder(r).Decode(&packet)
	if err != nil {
		return Packet{}, err
	}
	if err := packet.Validate(); err != nil {
		return Packet{}, err
	}
	return packet, nil
}

func DecodeLine(reader *bufio.Reader) (Packet, error) {
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return Packet{}, err
	}
	var packet Packet
	if err := json.Unmarshal(line, &packet); err != nil {
		return Packet{}, err
	}
	if err := packet.Validate(); err != nil {
		return Packet{}, err
	}
	return packet, nil
}
