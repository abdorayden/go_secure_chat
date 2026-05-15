package ui

import (
	"errors"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/widget"

	"chat_sec/internal/network"
	"chat_sec/internal/protocol"
	"chat_sec/internal/storage"
)

type Message struct {
	Text      string
	FromMe    bool
	Timestamp time.Time
	Status    string
}

type eventKind uint8

const (
	eventMessage eventKind = iota + 1
	eventAuth
	eventStatus
)

type inboundEvent struct {
	Kind    eventKind
	Message Message
	Status  string
	Err     string
}

type AppState struct {
	List          widget.List
	Input         widget.Editor
	UsernameInput widget.Editor
	EmailInput    widget.Editor
	PasswordInput widget.Editor

	SendBtn      widget.Clickable
	ConnectBtn   widget.Clickable
	ReconnectBtn widget.Clickable
	LoginTab     widget.Clickable
	SignupTab    widget.Clickable

	Incoming chan inboundEvent

	clientMu    sync.Mutex
	Client      *network.TransportClient
	clientToken uint64

	Messages   []Message
	Username   string
	StatusText string
	ErrorText  string
	Connected  bool
	Joined     bool
	SigningUp  bool
	Loading    bool
}

func newAppState() *AppState {
	return &AppState{
		List:          widget.List{List: layout.List{Axis: layout.Vertical}},
		Input:         widget.Editor{SingleLine: true, Submit: true},
		UsernameInput: widget.Editor{SingleLine: true, Submit: true},
		EmailInput:    widget.Editor{SingleLine: true, Submit: true},
		PasswordInput: widget.Editor{SingleLine: true, Submit: true, Mask: '*'},
		Incoming:      make(chan inboundEvent, 64),
		StatusText:    "Disconnected",
	}
}

func (s *AppState) Close() {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	if s.Client != nil {
		_ = s.Client.Close()
		s.Client = nil
	}
}

func (s *AppState) Connect(w *app.Window, addr string) error {
	s.clientMu.Lock()
	if s.Client != nil {
		_ = s.Client.Close()
	}
	client, err := network.Dial(addr)
	if err != nil {
		s.Client = nil
		s.Connected = false
		s.clientMu.Unlock()
		return err
	}
	s.Client = client
	s.clientToken++
	token := s.clientToken
	s.Connected = true
	s.StatusText = "Connected"
	s.ErrorText = ""
	s.clientMu.Unlock()

	go s.readLoop(w, client, token)
	return nil
}

func (s *AppState) readLoop(w *app.Window, client *network.TransportClient, token uint64) {
	for {
		packet, err := client.ReadPacket()
		if err != nil {
			s.emit(token, inboundEvent{Kind: eventStatus, Status: "Disconnected", Err: err.Error()})
			w.Invalidate()
			return
		}
		s.handlePacket(token, packet)
		w.Invalidate()
	}
}

func (s *AppState) handlePacket(token uint64, packet protocol.Packet) {
	switch packet.Type {
	case protocol.TypeSystem:
		if packet.Status == "auth_ok" {
			s.emit(token, inboundEvent{Kind: eventAuth, Status: packet.Username})
			return
		}
		s.emit(token, inboundEvent{
			Kind: eventMessage,
			Message: Message{
				Text:      packet.Payload,
				Timestamp: time.Unix(packet.Timestamp, 0),
				Status:    packet.Status,
			},
		})
	case protocol.TypePeerSync:
		s.clientMu.Lock()
		client := s.Client
		active := token == s.clientToken
		s.clientMu.Unlock()
		if !active || client == nil {
			return
		}
		if err := client.UpdatePeers(packet.Peers); err != nil {
			s.emit(token, inboundEvent{Kind: eventStatus, Status: "Peer sync failed", Err: err.Error()})
		}
	case protocol.TypeMessage:
		plaintext, err := s.decryptIncomingMessage(token, packet)
		if err != nil {
			s.emit(token, inboundEvent{Kind: eventStatus, Status: "Decrypt failed", Err: err.Error()})
			return
		}
		s.emit(token, inboundEvent{
			Kind: eventMessage,
			Message: Message{
				Text:      packet.Username + ": " + plaintext,
				FromMe:    packet.Username == s.Username,
				Timestamp: time.Unix(packet.Timestamp, 0),
				Status:    "delivered",
			},
		})
	case protocol.TypeError:
		s.emit(token, inboundEvent{Kind: eventStatus, Status: "Error", Err: packet.Error})
	}
}

func (s *AppState) decryptIncomingMessage(token uint64, packet protocol.Packet) (string, error) {
	s.clientMu.Lock()
	client := s.Client
	active := token == s.clientToken
	username := s.Username
	s.clientMu.Unlock()
	if !active || client == nil {
		return "", errors.New("stale client session")
	}
	return client.DecryptMessage(packet, username)
}

func (s *AppState) emit(token uint64, event inboundEvent) {
	s.clientMu.Lock()
	active := token == s.clientToken
	s.clientMu.Unlock()
	if !active {
		return
	}
	s.Incoming <- event
}

func (s *AppState) DrainEvents() {
	for {
		select {
		case event := <-s.Incoming:
			s.applyEvent(event)
		default:
			return
		}
	}
}

func (s *AppState) applyEvent(event inboundEvent) {
	switch event.Kind {
	case eventMessage:
		s.Messages = append(s.Messages, event.Message)
	case eventAuth:
		s.Joined = true
		s.Loading = false
		s.ErrorText = ""
		s.Username = event.Status
		s.StatusText = "Authenticated"
		s.EmailInput.SetText("")
		s.PasswordInput.SetText("")
	case eventStatus:
		s.Loading = false
		s.StatusText = event.Status
		s.ErrorText = event.Err
		if event.Status == "Disconnected" {
			s.Connected = false
			s.Joined = false
		}
	}
}

func (s *AppState) SubmitAuth() error {
	s.clientMu.Lock()
	client := s.Client
	s.clientMu.Unlock()
	if client == nil {
		return errors.New("not connected")
	}
	if err := s.ValidateAuthForm(); err != nil {
		return err
	}
	packet := protocol.Packet{
		Email:    strings.TrimSpace(s.EmailInput.Text()),
		Password: s.PasswordInput.Text(),
	}
	if s.SigningUp {
		packet.Type = protocol.TypeAuthSignup
		packet.Username = strings.TrimSpace(s.UsernameInput.Text())
	} else {
		packet.Type = protocol.TypeAuthLogin
	}
	return client.SendAuth(packet)
}

func (s *AppState) ValidateAuthForm() error {
	email := strings.TrimSpace(s.EmailInput.Text())
	password := s.PasswordInput.Text()
	if email == "" {
		return errors.New("email is required")
	}
	if password == "" {
		return errors.New("password is required")
	}
	if s.SigningUp {
		username := strings.TrimSpace(s.UsernameInput.Text())
		return storage.ValidateSignup(username, email, password)
	}
	if !strings.Contains(email, "@") {
		return errors.New("invalid email")
	}
	return nil
}

func (s *AppState) SwitchAuthMode(signingUp bool) {
	if s.SigningUp == signingUp {
		return
	}
	s.SigningUp = signingUp
	s.ErrorText = ""
	s.StatusText = "Connected"
	s.PasswordInput.SetText("")
	if !signingUp {
		s.UsernameInput.SetText("")
	}
}

func (s *AppState) SendMessage(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	s.clientMu.Lock()
	client := s.Client
	username := s.Username
	s.clientMu.Unlock()
	if client == nil {
		return errors.New("not connected")
	}
	packet, err := client.EncryptMessage(username, text)
	if err != nil {
		return err
	}
	if err := client.SendMessage(packet); err != nil {
		return err
	}
	s.Messages = append(s.Messages, Message{
		Text:      username + ": " + text,
		FromMe:    true,
		Timestamp: time.Now(),
		Status:    "sent",
	})
	s.Input.SetText("")
	return nil
}
