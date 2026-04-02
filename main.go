package main

import (
	"bufio"
	"encoding/base64"
	"image"
	"image/color"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"chat_sec/server/enc"
)

const (
	ServerPort = "9001"
	ServerHost = "localhost"
	label      = "Secure Chat"
)

var (
	connection   net.Conn = nil
	err          error    = nil
	serverPubKey *enc.RSAPublicKey
	keyMu        sync.RWMutex
)

type Message struct {
	Text   string
	FromMe bool
}

type AppState struct {
	Messages      []Message
	List          widget.List
	Input         widget.Editor
	UsernameInput widget.Editor
	Username      string
	Send          widget.Clickable
	JoinBtn       widget.Clickable
	Incoming      chan string
	Status        chan string
	Connected     bool
	StatusText    string
	Joined        bool
}

func main() {
	go func() {
		connection, err = net.Dial("tcp", ServerHost+":"+ServerPort)
		if err != nil {
			log.Fatalln(err)
		}

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

		keyMu.Lock()
		serverPubKey = serverPub
		keyMu.Unlock()

		log.Println("Received server public key")

		var w app.Window
		w.Option(
			app.Title(label),
			app.Size(unit.Dp(980), unit.Dp(640)),
		)
		if err := run(&w, reader); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window, reader *bufio.Reader) error {
	var ops op.Ops
	th := material.NewTheme()
	applyTheme(th)

	state := AppState{
		Messages:      []Message{},
		List:          widget.List{List: layout.List{Axis: layout.Vertical}},
		Input:         widget.Editor{SingleLine: true, Submit: true},
		UsernameInput: widget.Editor{SingleLine: true, Submit: true},
		Incoming:      make(chan string, 64),
		Status:        make(chan string, 8),
		Connected:     true,
		Joined:        false,
		StatusText:    "Enter username",
	}

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				state.Status <- "Disconnected"
				state.Connected = false
				w.Invalidate()
				return
			}
			msg := strings.TrimSpace(line)
			if msg == "" {
				continue
			}
			state.Incoming <- msg
			w.Invalidate()
		}
	}()

	for {
		e := w.Event()
		switch ev := e.(type) {
		case app.DestroyEvent:
			if connection != nil {
				_ = connection.Close()
			}
			return ev.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, ev)

			for {
				select {
				case msg := <-state.Incoming:
					state.Messages = append(state.Messages, Message{Text: msg, FromMe: false})
				default:
					goto messagesDrained
				}
			}
		messagesDrained:
			for {
				select {
				case status := <-state.Status:
					state.StatusText = status
				default:
					goto statusDrained
				}
			}
		statusDrained:

			if !state.Joined {
				usernameSubmitted := false

				for {
					ev, ok := state.UsernameInput.Update(gtx)
					if !ok {
						break
					}
					if _, ok := ev.(widget.SubmitEvent); ok {
						usernameSubmitted = true
					}
				}

				layoutJoinScreen(gtx, th, &state, w, &usernameSubmitted)
				if usernameSubmitted {
					username := strings.TrimSpace(state.UsernameInput.Text())
					if username == "" {
						username = "Anonymous"
					}
					state.Username = username
					state.UsernameInput.SetText("")

					_, err := connection.Write([]byte("USERNAME:" + username + "\n"))
					if err == nil {
						state.Joined = true
						state.StatusText = "Connected"
						state.Messages = append(state.Messages, Message{Text: "Joined as " + username, FromMe: false})
					}
				}
			} else {
				layoutChat(gtx, th, &state, w)
			}
			ev.Frame(gtx.Ops)
		}

	}
}

func layoutJoinScreen(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window, joinClicked *bool) layout.Dimensions {
	fill(gtx, th.Palette.Bg)

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))

				lbl := material.H3(th, "Enter Username")
				lbl.Color = th.Palette.Fg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))
				gtx.Constraints.Min.X = gtx.Constraints.Max.X

				macro := op.Record(gtx.Ops)
				dims := layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					editor := material.Editor(th, &state.UsernameInput, "Username")
					editor.TextSize = unit.Sp(18)
					editor.Color = th.Palette.Fg
					editor.HintColor = color.NRGBA{R: 0x7B, G: 0x84, B: 0x94, A: 0xFF}
					return editor.Layout(gtx)
				})
				call := macro.Stop()

				r := image.Rectangle{Max: dims.Size}
				rr := clip.RRect{Rect: r, NE: 8, NW: 8, SE: 8, SW: 8}
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, rr.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X = gtx.Dp(unit.Dp(300))
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(300))

				btn := material.Button(th, &state.JoinBtn, "Join Chat")
				btn.Background = th.Palette.ContrastBg
				btn.Color = th.Palette.ContrastFg
				btn.CornerRadius = unit.Dp(8)
				dims := btn.Layout(gtx)

				if state.JoinBtn.Clicked(gtx) {
					*joinClicked = true
				}
				return dims
			}),
		)
	})
}

func layoutChat(gtx layout.Context, th *material.Theme, state *AppState, w *app.Window) layout.Dimensions {
	fill(gtx, th.Palette.Bg)

	submittedText, submitted := consumeSubmit(gtx, state)
	var sendClicked bool
	layoutChatContent(gtx, th, state, &sendClicked)

	if submitted || sendClicked {
		text := strings.TrimSpace(state.Input.Text())
		if submitted && submittedText != "" {
			text = submittedText
		}
		if text != "" {
			state.Messages = append(state.Messages, Message{Text: text, FromMe: true})

			keyMu.RLock()
			pubKey := serverPubKey
			keyMu.RUnlock()

			if !state.Connected || pubKey == nil {
				state.Messages = append(state.Messages, Message{Text: "Not connected.", FromMe: false})
			} else {
				ciphertext, err := enc.Encrypt(pubKey, []byte(text))
				if err != nil {
					state.Messages = append(state.Messages, Message{Text: "Encryption failed: " + err.Error(), FromMe: false})
				} else {
					ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)
					_, err = connection.Write([]byte("MSG:" + ciphertextB64 + "\n"))
					if err != nil {
						state.Messages = append(state.Messages, Message{Text: "Send failed: " + err.Error(), FromMe: false})
						state.StatusText = "Disconnected"
						state.Connected = false
					}
				}
			}
			state.Input.SetText("")
		}
	}
	return layout.Dimensions{}
}

func applyTheme(th *material.Theme) {
	th.Palette = material.Palette{
		Bg:         color.NRGBA{R: 0xF3, G: 0xF5, B: 0xFA, A: 0xFF},
		Fg:         color.NRGBA{R: 0x16, G: 0x19, B: 0x22, A: 0xFF},
		ContrastBg: color.NRGBA{R: 0x2B, G: 0x6F, B: 0xFF, A: 0xFF},
		ContrastFg: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	}
	th.TextSize = unit.Sp(15)
}

func consumeSubmit(gtx layout.Context, state *AppState) (string, bool) {
	var submittedText string
	for {
		ev, ok := state.Input.Update(gtx)
		if !ok {
			break
		}
		if submit, ok := ev.(widget.SubmitEvent); ok {
			submittedText = strings.TrimSpace(submit.Text)
		}
	}
	return submittedText, submittedText != ""
}

func layoutChatContent(gtx layout.Context, th *material.Theme, state *AppState, sendClicked *bool) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return headerBar(gtx, th, state)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return messageList(gtx, th, state)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return inputBar(gtx, th, state, sendClicked)
		}),
	)
}

func headerBar(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	height := gtx.Dp(unit.Dp(56))
	gtx.Constraints.Min.Y = height
	gtx.Constraints.Max.Y = height
	fill(gtx, color.NRGBA{R: 0x12, G: 0x18, B: 0x2B, A: 0xFF})
	inset := layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(10), Bottom: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				userLabel := state.Username + " - " + label
				title := material.Label(th, unit.Sp(18), userLabel)
				title.Color = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
				return title.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				status := material.Label(th, unit.Sp(14), state.StatusText)
				if state.Connected {
					status.Color = color.NRGBA{R: 0xB8, G: 0xF1, B: 0xC2, A: 0xFF}
				} else {
					status.Color = color.NRGBA{R: 0xFF, G: 0xB3, B: 0xB3, A: 0xFF}
				}
				return status.Layout(gtx)
			}),
		)
	})
}

func messageList(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	inset := layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(12), Bottom: unit.Dp(12)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		maxWidth := gtx.Constraints.Max.X * 7 / 10
		return state.List.Layout(gtx, len(state.Messages), func(gtx layout.Context, i int) layout.Dimensions {
			msg := state.Messages[i]
			rowInset := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}
			return rowInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return messageRow(gtx, th, msg, maxWidth)
			})
		})
	})
}

func messageRow(gtx layout.Context, th *material.Theme, msg Message, maxWidth int) layout.Dimensions {
	bubble := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxWidth
		if msg.FromMe {
			return messageBubble(gtx, th, msg.Text, th.Palette.ContrastBg, th.Palette.ContrastFg)
		}
		return messageBubble(gtx, th, msg.Text, color.NRGBA{R: 0xE7, G: 0xEC, B: 0xF7, A: 0xFF}, th.Palette.Fg)
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

func messageBubble(gtx layout.Context, th *material.Theme, text string, bg, fg color.NRGBA) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body1(th, text)
		lbl.Color = fg
		return lbl.Layout(gtx)
	})
	call := macro.Stop()

	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: 12, NW: 12, SE: 12, SW: 12}
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func inputBar(gtx layout.Context, th *material.Theme, state *AppState, sendClicked *bool) layout.Dimensions {
	fill(gtx, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	inset := layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(10), Bottom: unit.Dp(10)}
	return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(
			gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return inputField(gtx, th, state)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(unit.Dp(88))
				btn := material.Button(th, &state.Send, "Send")
				btn.Background = th.Palette.ContrastBg
				btn.Color = th.Palette.ContrastFg
				btn.CornerRadius = unit.Dp(18)
				dims := btn.Layout(gtx)
				for state.Send.Clicked(gtx) {
					*sendClicked = true
				}
				return dims
			}),
		)
	})
}

func inputField(gtx layout.Context, th *material.Theme, state *AppState) layout.Dimensions {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		editor := material.Editor(th, &state.Input, "Type a message...")
		editor.TextSize = unit.Sp(15)
		editor.Color = th.Palette.Fg
		editor.HintColor = color.NRGBA{R: 0x7B, G: 0x84, B: 0x94, A: 0xFF}
		editor.LineHeight = unit.Sp(18)
		return editor.Layout(gtx)
	})
	call := macro.Stop()

	r := image.Rectangle{Max: dims.Size}
	rr := clip.RRect{Rect: r, NE: 14, NW: 14, SE: 14, SW: 14}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0xF4, G: 0xF6, B: 0xFB, A: 0xFF}, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

func fill(gtx layout.Context, c color.NRGBA) {
	r := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}
