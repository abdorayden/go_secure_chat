// Copyright (c) 2026 abdenour souane. All Rights Reserved.

package ui

import (
	"errors"
	"strconv"
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

type ThemeMode string

const (
	ThemeLight ThemeMode = "light"
	ThemeDark  ThemeMode = "dark"
)

type Message struct {
	Sender    string
	Body      string
	FromMe    bool
	Timestamp time.Time
	Status    string
}

type SidebarRoom struct {
	Name      string
	Badge     int
	Clickable widget.Clickable
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

type AuthState struct {
	UsernameInput  widget.Editor
	EmailInput     widget.Editor
	PasswordInput  widget.Editor
	ConnectBtn     widget.Clickable
	ReconnectBtn   widget.Clickable
	ThemeToggleBtn widget.Clickable
	LoginTab       widget.Clickable
	SignupTab      widget.Clickable
	SigningUp      bool
	Loading        bool
}

type ChatState struct {
	List     widget.List
	Input    widget.Editor
	SendBtn  widget.Clickable
	Messages []Message
}

type SessionState struct {
	Username   string
	Connected  bool
	Joined     bool
	StatusText string
	ErrorText  string
}

type ModalState struct {
	Visible        bool
	Kind           string
	Title          string
	Message        string
	ConfirmEditor  widget.Editor
	ConfirmBtn     widget.Clickable
	CancelBtn      widget.Clickable
	PrimaryAction  string
	SecondaryLabel string
}

type UIState struct {
	Theme               ThemeMode
	ActiveRoom          string
	Rooms               []SidebarRoom
	ThemeToggleBtn      widget.Clickable
	SettingsBtn         widget.Clickable
	NotificationsBtn    widget.Clickable
	EncryptionInfoBtn   widget.Clickable
	LogoutBtn           widget.Clickable
	DeleteAccountBtn    widget.Clickable
	SearchBtn           widget.Clickable
	InfoBtn             widget.Clickable
	AttachBtn           widget.Clickable
	SidebarWidth        float32
	EstimatedMemberText string
	Modal               ModalState
}

type AppState struct {
	Window  *app.Window
	Auth    AuthState
	Chat    ChatState
	Session SessionState
	UI      UIState

	Incoming chan inboundEvent

	clientMu    sync.Mutex
	Client      *network.TransportClient
	clientToken uint64
}

func newAppState(w *app.Window) *AppState {
	return &AppState{
		Window: w,
		Auth: AuthState{
			UsernameInput: widget.Editor{SingleLine: true, Submit: true},
			EmailInput:    widget.Editor{SingleLine: true, Submit: true},
			PasswordInput: widget.Editor{SingleLine: true, Submit: true, Mask: '*'},
		},
		Chat: ChatState{
			List:  widget.List{List: layout.List{Axis: layout.Vertical}},
			Input: widget.Editor{SingleLine: true, Submit: true},
		},
		Session: SessionState{
			StatusText: "Disconnected",
		},
		UI: UIState{
			Theme:               ThemeDark,
			ActiveRoom:          "General",
			SidebarWidth:        252,
			EstimatedMemberText: "0 members",
			Rooms: []SidebarRoom{
				{Name: "General", Badge: 2},
				{Name: "Team Alpha", Badge: 0},
				{Name: "Encrypted Lab", Badge: 1},
				{Name: "Future Channels", Badge: 0},
			},
		},
		Incoming: make(chan inboundEvent, 64),
	}
}

func (s *AppState) invalidate() {
	if s.Window != nil {
		s.Window.Invalidate()
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
		s.Session.Connected = false
		s.clientMu.Unlock()
		return err
	}
	s.Client = client
	s.clientToken++
	token := s.clientToken
	s.Session.Connected = true
	s.Session.StatusText = "Connected"
	s.Session.ErrorText = ""
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
				Sender:    "System",
				Body:      packet.Payload,
				Timestamp: time.Unix(packet.Timestamp, 0),
				Status:    statusLabel(packet.Status, "info"),
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
			return
		}
		s.UI.EstimatedMemberText = memberCountLabel(len(packet.Peers))
	case protocol.TypeMessage:
		plaintext, err := s.decryptIncomingMessage(token, packet)
		if err != nil {
			if packet.Status == "history" {
				return
			}
			s.emit(token, inboundEvent{Kind: eventStatus, Status: "Decrypt failed", Err: err.Error()})
			return
		}
		s.emit(token, inboundEvent{
			Kind: eventMessage,
			Message: Message{
				Sender:    packet.Username,
				Body:      plaintext,
				FromMe:    packet.Username == s.Session.Username,
				Timestamp: time.Unix(packet.Timestamp, 0),
				Status:    statusLabel(packet.Status, "delivered"),
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
	username := s.Session.Username
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
		s.Chat.Messages = append(s.Chat.Messages, event.Message)
	case eventAuth:
		s.Session.Joined = true
		s.Auth.Loading = false
		s.Session.ErrorText = ""
		s.Session.Username = event.Status
		s.Session.StatusText = "Authenticated"
		s.Auth.EmailInput.SetText("")
		s.Auth.PasswordInput.SetText("")
	case eventStatus:
		s.Auth.Loading = false
		s.Session.StatusText = event.Status
		s.Session.ErrorText = event.Err
		if event.Status == "Disconnected" {
			s.Session.Connected = false
			s.Session.Joined = false
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
	email := strings.TrimSpace(s.Auth.EmailInput.Text())
	packet := protocol.Packet{
		Email:    email,
		Password: s.Auth.PasswordInput.Text(),
	}
	if s.Auth.SigningUp {
		packet.Type = protocol.TypeAuthSignup
		packet.Username = strings.TrimSpace(s.Auth.UsernameInput.Text())
	} else {
		packet.Type = protocol.TypeAuthLogin
	}
	return client.SendAuth(packet)
}

func (s *AppState) ValidateAuthForm() error {
	email := strings.TrimSpace(s.Auth.EmailInput.Text())
	password := s.Auth.PasswordInput.Text()
	if email == "" {
		return errors.New("email is required")
	}
	if password == "" {
		return errors.New("password is required")
	}
	if s.Auth.SigningUp {
		username := strings.TrimSpace(s.Auth.UsernameInput.Text())
		return storage.ValidateSignup(username, email, password)
	}
	if !strings.Contains(email, "@") {
		return errors.New("invalid email")
	}
	return nil
}

func (s *AppState) SwitchAuthMode(signingUp bool) {
	if s.Auth.SigningUp == signingUp {
		return
	}
	s.Auth.SigningUp = signingUp
	s.Session.ErrorText = ""
	s.Session.StatusText = "Connected"
	s.Auth.PasswordInput.SetText("")
	if !signingUp {
		s.Auth.UsernameInput.SetText("")
	}
	s.invalidate()
}

func (s *AppState) ActivePalette() Palette {
	if s.UI.Theme == ThemeLight {
		return LightPalette
	}
	return DarkPalette
}

func (s *AppState) ToggleTheme() {
	if s.UI.Theme == ThemeDark {
		s.UI.Theme = ThemeLight
		s.invalidate()
		return
	}
	s.UI.Theme = ThemeDark
	s.invalidate()
}

func (s *AppState) HandleThemeToggle() {
	s.ToggleTheme()
}

func (s *AppState) SendMessage(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	s.clientMu.Lock()
	client := s.Client
	username := s.Session.Username
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
	s.Chat.Messages = append(s.Chat.Messages, Message{
		Sender:    username,
		Body:      text,
		FromMe:    true,
		Timestamp: time.Now(),
		Status:    "sent",
	})
	s.Chat.Input.SetText("")
	s.invalidate()
	return nil
}

func (s *AppState) Logout() {
	s.Close()
	s.Chat.Messages = nil
	s.Chat.Input.SetText("")
	s.Auth.UsernameInput.SetText("")
	s.Auth.EmailInput.SetText("")
	s.Auth.PasswordInput.SetText("")
	s.Session.Username = ""
	s.Session.Connected = false
	s.Session.Joined = false
	s.Session.StatusText = "Disconnected"
	s.Session.ErrorText = ""
	s.Auth.Loading = false
	s.UI.Modal = ModalState{}
	s.invalidate()
}

func (s *AppState) SetActiveRoom(name string) {
	if strings.TrimSpace(name) == "" || s.UI.ActiveRoom == name {
		return
	}
	s.UI.ActiveRoom = name
	s.invalidate()
}

func (s *AppState) OpenDeleteAccountModal() {
	s.UI.Modal = ModalState{
		Visible:        true,
		Kind:           "delete_account",
		Title:          "Delete Account",
		Message:        "Type DELETE to confirm. Backend account deletion is not implemented yet; this flow currently clears the local session after confirmation.",
		PrimaryAction:  "Delete Account",
		SecondaryLabel: "Cancel",
		ConfirmEditor:  widget.Editor{SingleLine: true, Submit: true},
	}
	s.invalidate()
}

func (s *AppState) OpenInfoModal(kind, title, message string) {
	s.UI.Modal = ModalState{
		Visible:       true,
		Kind:          kind,
		Title:         title,
		Message:       message,
		PrimaryAction: "Close",
	}
	s.invalidate()
}

func (s *AppState) CloseModal() {
	s.UI.Modal = ModalState{}
	s.invalidate()
}

func (s *AppState) ConfirmModal() {
	switch s.UI.Modal.Kind {
	case "delete_account":
		if strings.TrimSpace(strings.ToUpper(s.UI.Modal.ConfirmEditor.Text())) != "DELETE" {
			s.Session.ErrorText = "type DELETE to confirm account removal"
			s.invalidate()
			return
		}
		s.CloseModal()
		s.Logout()
	default:
		s.CloseModal()
	}
}

func statusLabel(status string, fallback string) string {
	if status == "" || status == "history" {
		return fallback
	}
	return status
}

func memberCountLabel(n int) string {
	if n == 1 {
		return "1 member"
	}
	return strconv.Itoa(n) + " members"
}
