package ui

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"chat_sec/internal/network"
	"chat_sec/internal/protocol"
)

const appLabel = "Secure Chat"

type Message struct {
	Text      string
	FromMe    bool
	Timestamp time.Time
	Status    string
}

type inboundEvent struct {
	Kind    string
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
	Client   *network.TransportClient

	Messages   []Message
	Username   string
	StatusText string
	ErrorText  string
	Connected  bool
	Joined     bool
	SigningUp  bool
	Loading    bool
}

func Run(addr string) error {
	var w app.Window
	w.Option(
		app.Title(appLabel),
		app.Size(unit.Dp(980), unit.Dp(640)),
	)

	state := AppState{
		List:          widget.List{List: layout.List{Axis: layout.Vertical}},
		Input:         widget.Editor{SingleLine: true, Submit: true},
		UsernameInput: widget.Editor{SingleLine: true, Submit: true},
		EmailInput:    widget.Editor{SingleLine: true, Submit: true},
		PasswordInput: widget.Editor{SingleLine: true, Submit: true, Mask: '*'},
		Incoming:      make(chan inboundEvent, 64),
		StatusText:    "Disconnected",
	}

	if err := connect(&state, &w, addr); err != nil {
		state.ErrorText = err.Error()
		state.StatusText = "Disconnected"
	}

	var ops op.Ops
	th := material.NewTheme()
	applyTheme(th)

	for {
		switch ev := w.Event().(type) {
		case app.DestroyEvent:
			if state.Client != nil {
				_ = state.Client.Close()
			}
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)
			drainEvents(&state)
			if !state.Joined {
				layoutAuth(gtx, th, &state, &w, addr)
			} else {
				layoutChat(gtx, th, &state)
			}
			ev.Frame(gtx.Ops)
		}
	}
}

func connect(state *AppState, w *app.Window, addr string) error {
	if state.Client != nil {
		_ = state.Client.Close()
	}
	client, err := network.Dial(addr)
	if err != nil {
		state.Connected = false
		return err
	}
	state.Client = client
	state.Connected = true
	state.StatusText = "Connected"
	state.ErrorText = ""
	go readLoop(state, w, client)
	return nil
}

func readLoop(state *AppState, w *app.Window, client *network.TransportClient) {
	for {
		packet, err := client.ReadPacket()
		if err != nil {
			state.Incoming <- inboundEvent{Kind: "status", Status: "Disconnected", Err: err.Error()}
			w.Invalidate()
			return
		}
		switch packet.Type {
		case protocol.TypeSystem:
			if packet.Status == "auth_ok" {
				state.Incoming <- inboundEvent{Kind: "auth", Status: packet.Username}
			} else {
				state.Incoming <- inboundEvent{
					Kind: "message",
					Message: Message{
						Text:      packet.Payload,
						Timestamp: time.Unix(packet.Timestamp, 0),
						Status:    packet.Status,
					},
				}
			}
		case protocol.TypePeerSync:
			if err := client.UpdatePeers(packet.Peers); err != nil {
				state.Incoming <- inboundEvent{Kind: "status", Status: "Peer sync failed", Err: err.Error()}
			}
		case protocol.TypeMessage:
			plaintext, err := client.DecryptMessage(packet, state.Username)
			if err != nil {
				state.Incoming <- inboundEvent{Kind: "status", Status: "Decrypt failed", Err: err.Error()}
				w.Invalidate()
				continue
			}
			state.Incoming <- inboundEvent{
				Kind: "message",
				Message: Message{
					Text:      packet.Username + ": " + plaintext,
					FromMe:    packet.Username == state.Username,
					Timestamp: time.Unix(packet.Timestamp, 0),
					Status:    "delivered",
				},
			}
		case protocol.TypeError:
			state.Incoming <- inboundEvent{Kind: "status", Status: "Error", Err: packet.Error}
		}
		w.Invalidate()
	}
}

func drainEvents(state *AppState) {
	for {
		select {
		case event := <-state.Incoming:
			switch event.Kind {
			case "message":
				state.Messages = append(state.Messages, event.Message)
			case "auth":
				state.Joined = true
				state.Loading = false
				state.ErrorText = ""
				state.Username = event.Status
				state.StatusText = "Authenticated"
			case "status":
				state.Loading = false
				state.StatusText = event.Status
				state.ErrorText = event.Err
				if event.Status == "Disconnected" {
					state.Connected = false
					state.Joined = false
				}
			}
		default:
			return
		}
	}
}

func layoutAuth(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window, addr string) layout.Dimensions {
	fill(gtx, th.Palette.Bg)
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return card(gtx, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H4(th, appLabel)
						title.Color = th.Palette.Fg
						return title.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return authTabs(gtx, th, state)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, &state.UsernameInput, "Username", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, &state.EmailInput, "Email", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return textField(gtx, th, &state.PasswordInput, "Password", true)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if state.ErrorText != "" {
							lbl := material.Body2(th, state.ErrorText)
							lbl.Color = color.NRGBA{R: 0xB0, G: 0x20, B: 0x20, A: 0xFF}
							return lbl.Layout(gtx)
						}
						lbl := material.Body2(th, state.StatusText)
						lbl.Color = color.NRGBA{R: 0x55, G: 0x62, B: 0x74, A: 0xFF}
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := "Log In"
						if state.SigningUp {
							label = "Sign Up"
						}
						btn := material.Button(th, &state.ConnectBtn, label)
						btn.Background = th.Palette.ContrastBg
						btn.CornerRadius = unit.Dp(10)
						dims := btn.Layout(gtx)
						for state.ConnectBtn.Clicked(gtx) {
							state.Loading = true
							state.ErrorText = ""
							if err := submitAuth(state); err != nil {
								state.Loading = false
								state.ErrorText = err.Error()
							}
						}
						return dims
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &state.ReconnectBtn, "Reconnect")
						btn.Background = color.NRGBA{R: 0xD8, G: 0xDF, B: 0xEB, A: 0xFF}
						btn.Color = th.Palette.Fg
						dims := btn.Layout(gtx)
						for state.ReconnectBtn.Clicked(gtx) {
							state.ErrorText = ""
							if err := connect(state, w, addr); err != nil {
								state.ErrorText = err.Error()
							}
						}
						return dims
					}),
				)
			})
		})
	})
}

func submitAuth(state *AppState) error {
	if state.Client == nil {
		return errors.New("not connected")
	}
	packet := protocol.Packet{
		Username: strings.TrimSpace(state.UsernameInput.Text()),
		Email:    strings.TrimSpace(state.EmailInput.Text()),
		Password: state.PasswordInput.Text(),
	}
	if state.SigningUp {
		packet.Type = protocol.TypeAuthSignup
	} else {
		packet.Type = protocol.TypeAuthLogin
	}
	return state.Client.SendAuth(packet)
}

func authTabs(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &state.LoginTab, "Login")
			if state.SigningUp {
				btn.Background = color.NRGBA{R: 0xDB, G: 0xE4, B: 0xF3, A: 0xFF}
				btn.Color = th.Palette.Fg
			}
			dims := btn.Layout(gtx)
			for state.LoginTab.Clicked(gtx) {
				state.SigningUp = false
				state.ErrorText = ""
			}
			return dims
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &state.SignupTab, "Sign Up")
			if !state.SigningUp {
				btn.Background = color.NRGBA{R: 0xDB, G: 0xE4, B: 0xF3, A: 0xFF}
				btn.Color = th.Palette.Fg
			}
			dims := btn.Layout(gtx)
			for state.SignupTab.Clicked(gtx) {
				state.SigningUp = true
				state.ErrorText = ""
			}
			return dims
		}),
	)
}

func layoutChat(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	fill(gtx, th.Palette.Bg)
	text, submitted := consumeSubmit(gtx, &state.Input)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return headerBar(gtx, th, state) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return messageList(gtx, th, state) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return inputBar(gtx, th, state, submitted, text)
		}),
	)
}

func inputBar(gtx layout.Context, th *material.Theme, state *AppState, submitted bool, text string) layout.Dimensions {
	fillWidth := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}
	dims := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return fillWidth.Layout(
			gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return textField(gtx, th, &state.Input, "Type a message", false)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, &state.SendBtn, "Send")
				btn.Background = th.Palette.ContrastBg
				dims := btn.Layout(gtx)
				clicked := false
				for state.SendBtn.Clicked(gtx) {
					clicked = true
				}
				if clicked || submitted {
					sendMessage(state, strings.TrimSpace(textOrCurrent(text, state.Input.Text())))
				}
				return dims
			}),
		)
	})
	return dims
}

func sendMessage(state *AppState, text string) {
	if text == "" || state.Client == nil {
		return
	}
	packet, err := state.Client.EncryptMessage(state.Username, text)
	if err != nil {
		state.ErrorText = err.Error()
		return
	}
	if err := state.Client.SendMessage(packet); err != nil {
		state.ErrorText = err.Error()
		return
	}
	state.Messages = append(state.Messages, Message{
		Text:      state.Username + ": " + text,
		FromMe:    true,
		Timestamp: time.Now(),
		Status:    "sent",
	})
	state.Input.SetText("")
}

func textOrCurrent(submitted string, current string) string {
	if submitted != "" {
		return submitted
	}
	return current
}

func consumeSubmit(gtx layout.Context, editor *widget.Editor) (string, bool) {
	for {
		ev, ok := editor.Update(gtx)
		if !ok {
			return "", false
		}
		if submit, ok := ev.(widget.SubmitEvent); ok {
			return submit.Text, true
		}
	}
}

func headerBar(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	height := gtx.Dp(unit.Dp(56))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	fill(gtx, color.NRGBA{R: 0x12, G: 0x18, B: 0x2B, A: 0xFF})
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(18), state.Username+" - "+appLabel)
				lbl.Color = color.NRGBA{A: 0xFF, R: 0xFF, G: 0xFF, B: 0xFF}
				return lbl.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, state.StatusText)
				if state.Connected {
					lbl.Color = color.NRGBA{R: 0xB8, G: 0xF1, B: 0xC2, A: 0xFF}
				} else {
					lbl.Color = color.NRGBA{R: 0xFF, G: 0xB3, B: 0xB3, A: 0xFF}
				}
				return lbl.Layout(gtx)
			}),
		)
	})
}

func messageList(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(12), Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		maxWidth := gtx.Constraints.Max.X * 7 / 10
		return state.List.Layout(gtx, len(state.Messages), func(gtx layout.Context, i int) layout.Dimensions {
			msg := state.Messages[i]
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return messageRow(gtx, th, msg, maxWidth)
			})
		})
	})
}

func messageRow(gtx layout.Context, th *material.Theme, msg Message, maxWidth int) layout.Dimensions {
	bubble := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxWidth
		bg := color.NRGBA{R: 0xE7, G: 0xEC, B: 0xF7, A: 0xFF}
		fg := color.NRGBA{R: 0x16, G: 0x19, B: 0x22, A: 0xFF}
		if msg.FromMe {
			bg = color.NRGBA{R: 0x2B, G: 0x6F, B: 0xFF, A: 0xFF}
			fg = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
		}
		return messageBubble(gtx, th, msg, bg, fg)
	}
	if msg.FromMe {
		return layout.Flex{Axis: layout.Horizontal}.Layout(
			gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
			layout.Rigid(bubble),
		)
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(
		gtx,
		layout.Rigid(bubble),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
	)
}

func messageBubble(gtx layout.Context, th *material.Theme, msg Message, bg, fg color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, msg.Text)
				lbl.Color = fg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				meta := material.Caption(th, msg.Timestamp.Format("15:04:05")+"  "+msg.Status)
				meta.Color = fg
				return meta.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: 12, NW: 12, SE: 12, SW: 12}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func textField(gtx layout.Context, th *material.Theme, editor *widget.Editor, hint string, fullWidth bool) layout.Dimensions {
	if fullWidth {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	}
	return card(gtx, color.NRGBA{R: 0xF4, G: 0xF6, B: 0xFB, A: 0xFF}, func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			w := material.Editor(th, editor, hint)
			w.Color = th.Palette.Fg
			w.HintColor = color.NRGBA{R: 0x7B, G: 0x84, B: 0x94, A: 0xFF}
			return w.Layout(gtx)
		})
	})
}

func card(gtx layout.Context, bg color.NRGBA, child layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := child(gtx)
	call := macro.Stop()
	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: 14, NW: 14, SE: 14, SW: 14}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func applyTheme(th *material.Theme) {
	th.Palette = material.Palette{
		Bg:         color.NRGBA{R: 0xF3, G: 0xF5, B: 0xFA, A: 0xFF},
		Fg:         color.NRGBA{R: 0x16, G: 0x19, B: 0x22, A: 0xFF},
		ContrastBg: color.NRGBA{R: 0x2B, G: 0x6F, B: 0xFF, A: 0xFF},
		ContrastFg: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	}
}

func fill(gtx layout.Context, c color.NRGBA) {
	r := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}
